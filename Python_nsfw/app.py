import psycopg2
import psycopg2.extras
import time
import logging
import requests
import os
import tempfile
import json
from urllib.parse import urlparse
from nsfw_detector import load_model, classify
from typing import List, Tuple, Optional, Dict
from psycopg2.pool import ThreadedConnectionPool
from http.server import HTTPServer, BaseHTTPRequestHandler
import threading


class HTTPRequestHandler(BaseHTTPRequestHandler):
    def __init__(self, *args, processor=None, **kwargs):
        self.processor = processor
        super().__init__(*args, **kwargs)
    
    def do_GET(self):
        if self.path == '/status':
            try:
                self.send_response(200)
                self.send_header('Content-type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps({"status": "running"}).encode())
            except Exception as e:
                self.send_response(500)
                self.send_header('Content-type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps({"error": str(e)}).encode())

class NSFWDatabaseProcessor:
    def __init__(self, db_config, model_path='mobilenet_v2_140_224'):
        """Initialize processor with database config and model path"""
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s'
        )
        self.db_config = db_config
        self.model_path = model_path
        self.model = None
        self.temp_dir = tempfile.mkdtemp()
        self.stop_event = False
        
        # Initialize connection pool with minimal connections
        self.pool = ThreadedConnectionPool(
            minconn=1,
            maxconn=2,  # One for main process, one for HTTP server
            **db_config
        )
        logging.info("Initialized connection pool")
        
        try:
            import PIL
            logging.info(f"PIL version: {PIL.__version__}")
        except ImportError:
            logging.error("PIL not found. Installing Pillow...")
            os.system('pip install Pillow')
        
        logging.info(f"Initialized NSFW processor. Temp directory: {self.temp_dir}")

    def get_connection(self):
        """Get a connection from the pool"""
        return self.pool.getconn()

    def return_connection(self, conn):
        """Return a connection to the pool"""
        self.pool.putconn(conn)

    def download_image(self, url: str) -> Optional[str]:
        """Download and validate image from URL"""
        try:
            if 'drive.google.com' in url:
                file_id = url.split('id=')[-1]
                url = f"https://drive.google.com/uc?export=download&id={file_id}"
            
            response = requests.get(url, timeout=60, stream=True)
            response.raise_for_status()
            
            content_type = response.headers.get('content-type', '')
            if not content_type.startswith('image/'):
                logging.error(f"Invalid content type: {content_type}")
                return None
            
            file_name = f"temp_{int(time.time())}_{os.path.basename(urlparse(url).path)}"
            if not file_name.lower().endswith(('.png', '.jpg', '.jpeg', '.gif', '.bmp')):
                file_name += '.jpg'
            
            temp_path = os.path.join(self.temp_dir, file_name)
            
            with open(temp_path, 'wb') as f:
                f.write(response.content)
            
            return temp_path
            
        except Exception as e:
            logging.error(f"Failed to download image from {url}: {e}")
            return None

    def get_pending_images(self) -> List[Tuple[int, str]]:
        """Get pending images from database including stalled jobs"""
        conn = self.get_connection()
        try:
            with conn.cursor() as cur:
                cur.execute("""
                    SELECT id, download_url 
                    FROM images 
                    WHERE (
                        -- New unprocessed images
                        (is_approved = 0 
                        AND marked_for_review = 0
                        AND processing_started IS NULL)
                        OR
                        -- Stalled jobs (started but not completed after 5 minutes)
                        (processing_started < NOW() - INTERVAL '5 minutes'
                        AND processing_completed IS NULL)
                    )
                    LIMIT 1
                    FOR UPDATE SKIP LOCKED
                """)
                
                images = cur.fetchall()
                
                if images:
                    image_ids = [img[0] for img in images]
                    cur.execute("""
                        UPDATE images 
                        SET processing_started = NOW() 
                        WHERE id = ANY(%s)
                    """, (image_ids,))
                    conn.commit()
                    
                return images
        finally:
            self.return_connection(conn)

    def process_image_results(self, results: Dict) -> Dict:
        """Process and validate classification results"""
        try:
            if isinstance(results, dict):
                if len(results) == 0:
                    raise ValueError("Empty results dictionary")
                    
                if all(isinstance(k, str) and os.path.sep in k for k in results.keys()):
                    result_dict = list(results.values())[0]
                else:
                    result_dict = results
                    
                required_keys = {'porn', 'sexy', 'hentai'}
                if not all(key in result_dict for key in required_keys):
                    raise ValueError(f"Missing required keys. Found: {list(result_dict.keys())}")
                
                return result_dict
            else:
                raise ValueError(f"Unexpected results type: {type(results)}")
                
        except Exception as e:
            logging.error(f"Error processing results: {e}")
            raise

    def update_image_status(self, image_id: int, results: Dict):
        """Update image processing results in database"""
        conn = self.get_connection()
        try:
            with conn.cursor() as cur:
                try:
                    result_dict = self.process_image_results(results)
                    
                    max_nsfw_score = max(
                        result_dict['porn'],
                        result_dict['sexy'],
                        result_dict['hentai']
                    )
                    
                    if max_nsfw_score <= 0.30:
                        is_approved = 1
                        needs_review = 0
                    elif max_nsfw_score <= 0.75:
                        is_approved = 0
                        needs_review = 1
                    else:
                        is_approved = 2
                        needs_review = 0
                    
                except Exception as e:
                    logging.error(f"Error processing results for image {image_id}: {e}")
                    is_approved = 0
                    needs_review = 1
                    result_dict = {'error': str(e)}
                
                cur.execute("""
                    UPDATE images 
                    SET is_approved = %s,
                        marked_for_review = %s,
                        processing_completed = NOW()
                    WHERE id = %s
                    """, (is_approved, needs_review, image_id))
                conn.commit()
                
                logging.info(f"Image {image_id} processed - Status: {is_approved}")
                if 'error' not in result_dict:
                    logging.info(f"Scores - Porn: {result_dict['porn']:.3f}, "
                               f"Sexy: {result_dict['sexy']:.3f}, "
                               f"Hentai: {result_dict['hentai']:.3f}")
                
        except Exception as e:
            logging.error(f"Database error for image {image_id}: {e}")
            conn.rollback()
        finally:
            self.return_connection(conn)

    def process_single_image(self, image_data: Tuple[int, str]):
        """Process a single image with error handling"""
        image_id, image_url = image_data
        temp_path = None
        
        try:
            logging.info(f"Processing image {image_id}: {image_url}")
            
            temp_path = self.download_image(image_url)
            if not temp_path:
                raise ValueError("Failed to download or validate image")

            results = classify(self.model, temp_path)
            self.update_image_status(image_id, results)
            
        except Exception as e:
            logging.error(f"Error processing image {image_id}: {e}")
            conn = self.get_connection()
            try:
                with conn.cursor() as cur:
                    cur.execute("""
                        UPDATE images 
                        SET is_approved = 0,
                            marked_for_review = 1,
                            processing_completed = NOW()
                        WHERE id = %s
                        """, (image_id,))
                conn.commit()
            except Exception as db_error:
                logging.error(f"Failed to update error status: {db_error}")
                conn.rollback()
            finally:
                self.return_connection(conn)
        finally:
            if temp_path and os.path.exists(temp_path):
                try:
                    os.remove(temp_path)
                except Exception as e:
                    logging.error(f"Error removing temporary file {temp_path}: {e}")

    def process_images(self):
        """Main processing function - single threaded"""
        if self.model is None:
            from nsfw_detector import load_model, classify
            self.model = load_model(self.model_path)
            logging.info("Model loaded successfully")

        while not self.stop_event:
            try:
                pending_images = self.get_pending_images()
                
                if not pending_images:
                    logging.info("No pending images found. Sleeping for 5 seconds...")
                    time.sleep(5)
                    continue

                logging.info(f"Processing image {pending_images[0][0]}")
                self.process_single_image(pending_images[0])
                    
            except Exception as e:
                logging.error(f"Processing error: {e}")
                time.sleep(5)

    def cleanup(self):
        """Clean up resources"""
        try:
            self.stop_event = True
            
            # Close all database connections in the pool
            self.pool.closeall()
            logging.info("Closed all database connections")
            
            # Clean up temporary directory
            for file in os.listdir(self.temp_dir):
                file_path = os.path.join(self.temp_dir, file)
                try:
                    os.remove(file_path)
                except Exception as e:
                    logging.error(f"Error removing file {file_path}: {e}")
            os.rmdir(self.temp_dir)
            logging.info("Cleaned up temporary directory")
        except Exception as e:
            logging.error(f"Failed to cleanup resources: {e}")

    def start_http_server(self):
        """Start HTTP server on port 8000"""
        def handler(*args):
            HTTPRequestHandler(*args, processor=self)
        
        server = HTTPServer(('', 8000), handler)
        server_thread = threading.Thread(target=server.serve_forever)
        server_thread.daemon = True
        server_thread.start()
        logging.info("HTTP server started on port 8000")

def main():
    db_config = {
        'dbname': 'a',
        'user': 'a',
        'password': 'a',
        'host': 'a',
        'port': '5432'
    }

    processor = NSFWDatabaseProcessor(db_config)
    
    try:
        processor.start_http_server()
        processor.process_images()
    except KeyboardInterrupt:
        logging.info("Service stopped by user")
        processor.cleanup()
    except Exception as e:
        logging.error(f"Service error: {e}")
        processor.cleanup()

if __name__ == "__main__":
    main()