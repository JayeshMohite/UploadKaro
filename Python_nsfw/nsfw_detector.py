import argparse
import json
import os
from os import listdir
from os.path import isfile, join, exists, isdir, abspath

import numpy as np
import tensorflow as tf
from tensorflow import keras

def setup_device():
    """Configure TensorFlow for minimal memory usage on CPU"""
    try:
        # Configure CPU-only mode
        os.environ['CUDA_VISIBLE_DEVICES'] = '-1'
        tf.config.set_visible_devices([], 'GPU')
        
        # Reduce thread parallelism
        tf.config.threading.set_inter_op_parallelism_threads(1)
        tf.config.threading.set_intra_op_parallelism_threads(1)
    except Exception as e:
        print(f"Device configuration warning (non-critical): {e}")

def load_model(model_path, batch_size=1):
    """Load model with broader compatibility across TF versions"""
    if model_path is None or not os.path.exists(model_path):
        raise ValueError("saved_model_path must be the valid directory of a saved model to load.")
    
    # Setup device configuration
    setup_device()
    
    try:
        print(f"Loading model from: {model_path}")
        
        # Clear any existing models
        tf.keras.backend.clear_session()
        
        # Try loading as SavedModel format first
        if os.path.isdir(model_path):
            try:
                base_model = tf.saved_model.load(model_path)
                
                # Create a wrapper model
                class SavedModelWrapper(keras.Model):
                    def __init__(self, saved_model):
                        super().__init__()
                        self.saved_model = saved_model
                        self.rescale = keras.layers.Rescaling(scale=1./255)
                    
                    def call(self, inputs):
                        x = self.rescale(inputs)
                        if hasattr(self.saved_model, 'signatures'):
                            return self.saved_model.signatures['serving_default'](tf.cast(x, tf.float32))
                        return self.saved_model(tf.cast(x, tf.float32))
                
                model = SavedModelWrapper(base_model)
                # Force model to build
                model.build((batch_size, 224, 224, 3))
                print("SavedModel loaded successfully")
                return model
                
            except Exception as e:
                print(f"SavedModel loading failed: {e}")
                # Fall through to alternative loading methods
        
        # Try loading as Keras model (H5 format)
        if model_path.endswith('.h5'):
            try:
                model = keras.models.load_model(model_path)
                # Add rescaling if not present
                if not any(isinstance(layer, keras.layers.Rescaling) for layer in model.layers):
                    model = keras.Sequential([
                        keras.layers.Input(shape=(224, 224, 3)),
                        keras.layers.Rescaling(scale=1./255),
                        model
                    ])
                model.build((batch_size, 224, 224, 3))
                print("H5 model loaded successfully")
                return model
            except Exception as e:
                print(f"H5 loading failed: {e}")
        
        raise ValueError("Failed to load model using any available method")
            
    except Exception as e:
        raise ValueError(f"Model loading failed: {e}")

def load_images(image_paths, image_size, batch_size=8, verbose=True):
    """Memory-efficient image loading with small batches"""
    def image_generator():
        if isdir(image_paths):
            parent = abspath(image_paths)
            paths = [join(parent, f) for f in listdir(image_paths) if isfile(join(parent, f))]
        elif isfile(image_paths):
            paths = [image_paths]
        
        current_batch = []
        current_paths = []
        
        for img_path in paths:
            try:
                if verbose:
                    print(f"Loading image: {img_path}")
                image = keras.utils.load_img(img_path, target_size=image_size)
                image = keras.utils.img_to_array(image)
                current_batch.append(image)
                current_paths.append(img_path)
                
                if len(current_batch) >= batch_size:
                    yield np.asarray(current_batch), current_paths
                    current_batch = []
                    current_paths = []
                    
                    # Clear memory after each batch
                    tf.keras.backend.clear_session()
                    
            except Exception as ex:
                print(f"Failed to load image {img_path}: {ex}")
        
        if current_batch:
            yield np.asarray(current_batch), current_paths
    
    return image_generator()

def classify(model, input_paths, image_dim=224, batch_size=8, predict_args={}):
    """Memory-efficient classification using small batches"""
    results = {}
    
    try:
        for images, image_paths in load_images(input_paths, (image_dim, image_dim), batch_size):
            batch_probs = classify_nd(model, images, predict_args)
            results.update(dict(zip(image_paths, batch_probs)))
            
            # Clear memory after each batch
            tf.keras.backend.clear_session()
            
        return results
    except Exception as e:
        print(f"Classification failed: {e}")
        raise

def classify_nd(model, nd_images, predict_args={}):
    """Memory-efficient classification implementation"""
    try:
        model_preds = model.predict(nd_images, **predict_args)
        
        # Handle different output formats from SavedModel
        if isinstance(model_preds, dict):
            # Get the actual predictions from the SavedModel output
            model_preds = next(iter(model_preds.values()))
        
        categories = ['drawings', 'hentai', 'neutral', 'porn', 'sexy']
        
        probs = []
        for single_preds in model_preds:
            single_probs = {}
            for j, pred in enumerate(single_preds):
                single_probs[categories[j]] = float(pred)
            probs.append(single_probs)
            
        return probs
    except Exception as e:
        print(f"Prediction failed: {e}")
        raise

def main(args=None):
    """Memory-optimized main function"""
    parser = argparse.ArgumentParser(
        description="Memory-optimized NSFW classification of images"
    )
    
    parser.add_argument('--image_source', type=str, required=True,
                       help='A directory of images or a single image to classify')
    parser.add_argument('--saved_model_path', type=str, required=True,
                       help='The model to load')
    parser.add_argument('--image_dim', type=int, default=224,
                       help="The square dimension of the model's input shape")
    parser.add_argument('--batch_size', type=int, default=8,
                       help="Batch size for processing images")
    
    try:
        config = vars(parser.parse_args(args) if args is not None else parser.parse_args())
        
        if not exists(config['image_source']):
            raise ValueError("Image source does not exist")
        
        model = load_model(config['saved_model_path'], batch_size=config['batch_size'])
        image_preds = classify(
            model, 
            config['image_source'], 
            config['image_dim'],
            config['batch_size']
        )
        print(json.dumps(image_preds, indent=2), '\n')
        
    except Exception as e:
        print(f"Error in main execution: {e}")
        raise

if __name__ == "__main__":
    main()