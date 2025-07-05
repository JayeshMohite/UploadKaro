let currentPage = 0;
let pageSize = 6;
let orderBy = "created_at"; // Default order by
let isLoading = false;
let selectedFiles = new Set();
let pageHistory = [];
let lastScrollTime = new Date();
let totalPages = 0;
const throttleDelay = 5000; // 5 seconds
const floatingUploadButton = document.getElementById("floatingUploadButton");
let imagesData = [];
let viewType = "listView";

function getUrl(endpoint) {
  if (Math.round(Math.random())) {
    return (
      "https://a/" + endpoint
    );
  } else {
    return (
      "https://a/" + endpoint
    );
  }
}
// Initialize drag and drop functionality
function initializeDragAndDrop() {
  const uploadZone = document.getElementById("uploadZone");
  const fileInput = document.getElementById("images");

  ["dragenter", "dragover", "dragleave", "drop"].forEach((eventName) => {
    uploadZone.addEventListener(eventName, preventDefaults, false);
  });

  function preventDefaults(e) {
    e.preventDefault();
    e.stopPropagation();
  }

  ["dragenter", "dragover"].forEach((eventName) => {
    uploadZone.addEventListener(eventName, () => {
      uploadZone.classList.add("drag-over");
    });
  });

  ["dragleave", "drop"].forEach((eventName) => {
    uploadZone.addEventListener(eventName, () => {
      uploadZone.classList.remove("drag-over");
    });
  });

  uploadZone.addEventListener("drop", handleDrop);
  fileInput.addEventListener("change", handleFileSelect);
}

// Handle dropped files
function handleDrop(e) {
  const dt = e.dataTransfer;
  const files = dt.files;
  handleFiles(files);
}

// Handle selected files from input
function handleFileSelect(e) {
  const files = e.target.files;
  handleFiles(files);
}

// Helper object for image type validation
const ImageTypes = {
  // Common web formats
  standard: [".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg"],
  // Camera/Professional formats
  camera: [".heic", ".heif", ".raw", ".cr2", ".nef", ".arw", ".dng"],
  // Other supported formats
  other: [".tiff", ".tif", ".psd", ".ico"],
};

// Consolidated list of valid MIME types
const validMimeTypes = [
  "image/", // Standard images
  "image/heic", // HEIC format
  "image/heif", // HEIF format
  "heic",
  "heif",
];

// Function to check if file is a valid image
function isValidImageFile(file) {
  // Get file extension
  const ext = "." + file.name.split(".").pop().toLowerCase();

  // Check if extension is in our supported list
  const isValidExt = [
    ...ImageTypes.standard,
    ...ImageTypes.camera,
    ...ImageTypes.other,
  ].includes(ext);

  console.log("file type:", file.type);

  // Check MIME type or use extension as a fallback if MIME type is missing
  const isValidType = validMimeTypes.some(
    (type) =>
      file.type.startsWith(type) ||
      (!file.type && isValidExt) || // Use extension if MIME type is missing
      (file.type === "application/octet-stream" && isValidExt)
  );

  // Size validation (50MB limit)
  const isValidSize = file.size <= 50 * 1024 * 1024;

  return {
    isValid: isValidType && isValidSize,
    error: !isValidType
      ? "Unsupported format"
      : !isValidSize
      ? "File exceeds 50MB limit"
      : null,
  };
}

// Enhanced file handling function
function handleFiles(files) {
  console.log("Processing files:", files);

  const validFiles = Array.from(files).filter((file) => {
    const validation = isValidImageFile(file);

    if (!validation.isValid) {
      showToast(`${file.name}: ${validation.error}`);
      console.log(`Rejected file: ${file.name} - ${validation.error}`);
    } else {
      console.log(`Accepted file: ${file.name} (${file.type})`);
    }

    return validation.isValid;
  });

  validFiles.forEach((file) => {
    if (!selectedFiles.has(file.name)) {
      selectedFiles.add(file.name);
      previewFile(file);
    }
  });
}

// Create preview for selected file
function previewFile(file) {
  const selectedFilesContainer = document.getElementById("selectedFiles");

  // Create preview container
  const preview = document.createElement("div");
  preview.className = "file-preview";

  // Show loading state
  preview.innerHTML = `
        <div class="preview-loading">Loading preview...</div>
        <div class="file-info">
            <span class="file-name">${file.name}</span>
        </div>
        <div class="remove-file"  data-filename="${file.name}"><img  src="https://www.svgrepo.com/show/521590/cross.svg"></div>
    `;

  selectedFilesContainer.appendChild(preview);

  // Handle preview based on file type
  if (
    file.type.startsWith("image/") ||
    file.name.toLowerCase().endsWith(".heic") ||
    file.name.toLowerCase().endsWith(".heif")
  ) {
    const reader = new FileReader();

    reader.onload = function (e) {
      // For special formats that might not preview well, show a placeholder
      const isSpecialFormat = /\.(heic|heif|raw|cr2|nef|arw|dng)$/i.test(
        file.name
      );

      if (isSpecialFormat) {
        preview.innerHTML = `
                    <div class="special-format-preview">
                        <div class="format-icon">📸</div>
                        <div class="format-info">
                            <span class="format-name">${file.name
                              .split(".")
                              .pop()
                              .toUpperCase()} Image</span>
                        </div>
                    </div>
                    <div class="remove-file"  data-filename="${
                      file.name
                    }"><img  src="https://www.svgrepo.com/show/521590/cross.svg"></div>
                `;
      } else {
        preview.innerHTML = `
                    <img src="${e.target.result}" alt="${file.name}">
                    <div class="file-info">
                        <span class="file-name">${file.name}</span>
                    </div>
                    <div class="remove-file"  data-filename="${file.name}"><img  src="https://www.svgrepo.com/show/521590/cross.svg"></div>
                `;
      }
    };

    reader.onerror = function () {
      preview.innerHTML = `
                <div class="preview-error">Error loading preview</div>
                <div class="file-info">
                    <span class="file-name">${file.name}</span>
                </div>
                <div class="remove-file"  data-filename="${file.name}"><img  src="https://www.svgrepo.com/show/521590/cross.svg"></div>
            `;
    };

    reader.readAsDataURL(file);
  }

  // Add remove functionality
  preview.querySelector(".remove-file").addEventListener("click", function () {
    selectedFiles.delete(this.dataset.filename);
    preview.remove();
  });
}

// Enhanced upload function with progress tracking
async function uploadFiles(files) {
  const formData = new FormData();
  const progressBar = document.getElementById("uploadProgress");
  const progressBarFill = document.getElementById("progressBarFill");

  // Add files to FormData
  Array.from(files).forEach((file) => {
    const validation = isValidImageFile(file);
    if (validation.isValid) {
      // Handle special file types
      if (file.type === "application/octet-stream") {
        // Get extension
        const ext = file.name.split(".").pop().toLowerCase();
        // Create appropriate MIME type
        const mimeType =
          ext === "heic"
            ? "image/heic"
            : ext === "heif"
            ? "image/heif"
            : ext === "raw"
            ? "image/raw"
            : `image/${ext}`;

        // Create new blob with correct MIME type
        const blob = new Blob([file], { type: mimeType });
        formData.append("images", blob, file.name);
      } else {
        // Normal file append
        formData.append("images", file);
      }
      formData.append("username", localStorage.getItem("userName"));
    }
  });

  progressBar.classList.add("active");
  progressBarFill.style.width = "0%";

  try {
    const response = await fetch(getUrl("image/upload"), {
      method: "POST",
      body: formData,
      onUploadProgress: (progressEvent) => {
        const percentCompleted = Math.round(
          (progressEvent.loaded * 100) / progressEvent.total
        );
        progressBarFill.style.width = percentCompleted + "%";
      },
    });

    if (response.ok) {
      const text = await response.text();
      showToast(text);
      loadImages();

      // Clear selected files
      selectedFiles.clear();
      document.getElementById("selectedFiles").innerHTML = "";
      document.getElementById("images").value = "";
    } else {
      showToast("Upload failed.");
    }
  } catch (error) {
    console.error("Upload Error:", error);
    showToast("Upload error occurred.");
  } finally {
    setTimeout(() => {
      progressBar.classList.remove("active");
      progressBarFill.style.width = "0%";
    }, 1000);
  }
}

// Toast function to show upload status message
function showToast(message) {
  const toast = document.createElement("div");
  toast.className = "toast";
  toast.textContent = message;
  document.body.appendChild(toast);

  setTimeout(() => {
    toast.classList.add("show");
  }, 100);

  setTimeout(() => {
    toast.classList.remove("show");
    setTimeout(() => document.body.removeChild(toast), 300);
  }, 3000);
}

// Function to handle file download
function downloadFile(url, filename) {
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
}

// Function to show loading state
function setLoading(loading) {
  const imageList = document.getElementById("imageList");
  if (loading) {
    imageList.classList.add("loading");
    isLoading = loading;
  } else {
    imageList.classList.remove("loading");
    isLoading = loading;
  }
}

// Function to handle sort change
function handleSortChange(value) {
  if (orderBy === value) return;
  orderBy = value;

  // Reset pagination and reload images
  currentPage = 0;
  pageHistory = [];

  const imageList = document.getElementById("imageList");

  if (imageList) {
    imageList.innerHTML = ""; // Clear current images
  }
  loadImages();
}

function handleViewChange(value) {
  viewType = value;

  // Reset pagination and reload images
  const imageList = document.getElementById("imageList");
  if (imageList) {
    if (viewType === "gridView") {
      imageList.classList.replace("list-view", "grid-view");
    } else {
      imageList.classList.replace("grid-view", "list-view");
    }
  }
}
// Function to load images with infinite scroll
async function loadImages() {
  if (isLoading) return;

  setLoading(true);
  const container = document.getElementsByClassName("container")[0];
  const intersectionObserver = document.getElementById("intersection-observer");
  try {
    container.removeChild(intersectionObserver);
    const response = await fetch(getUrl("image/listing"), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        page_size: pageSize,
        page_number: currentPage,
        order_by: orderBy,
      }),
    });

    const data = await response.json();
    const imageList = document.getElementById("imageList");

    if (viewType === "gridView") {
      imageList.classList.replace("list-view", "grid-view");
    } else {
      imageList.classList.replace("grid-view", "list-view");
    }

    data.files.forEach((file) => {
      const div = document.createElement("div");
      div.className = "image-item";
      div.innerHTML = `
                  <img src="${file.thumbnail}" alt="${file.name}" loading="lazy" data-file-id="${file.id}">
                  <div class="name">${file.name}</div>
                  <div class="like-container">
                  <button class="download-link" data-url="${file.download_url}" data-filename="${file.name}">Download</button>
                  <div class="like-group">
                     <button class="share-link" data-file-id="${file.id}">🔗</button>
                   <button class="like-button" onclick="toggleLike('${file.id}')">❤️</button>
                  <span class="like-count" id="like-count-${file.id}">${file.liked_count}</span>
                  </div>
                  </div>
              `;
      // Append the new images to the end of the existing list
      imageList.appendChild(div);
    });

    // Update navigation state
    if (data.total_pages > currentPage) {
      pageHistory.push(currentPage);
      currentPage++;
    } else if (pageHistory.length >= 2) {
      pageHistory.pop();
      currentPage--;
    } else {
      pageHistory = [];
      currentPage = 1;
    }

    totalPages = data.total_pages;
  } catch (error) {
    console.error("Error loading images:", error);
    showToast("No images found.");
  } finally {
    setLoading(false);
    container.appendChild(intersectionObserver);
  }
}

const options = {
  root: null,
  rootMargin: "0px",
  threshold: 1.0,
};

const observer = new IntersectionObserver(handleInfiniteScroll, options);

observer.observe(document.getElementById("intersection-observer"));

function handleInfiniteScroll(entries, observer) {
  const currentTime = Date.now();
  const timeElapsed = currentTime - lastScrollTime;
  if (timeElapsed >= throttleDelay && currentPage < totalPages && !isLoading)
    loadImages();
}

// Add an event listener for the floating button click
floatingUploadButton.addEventListener("click", () => {
  const fileInput = document.getElementById("images");
  fileInput.click(); // Trigger the file input click
});

// Pagination handlers
function handlePrev() {
  if (pageHistory.length > 1) {
    currentPage = pageHistory.pop();
    loadImages();
  }
}

function handleNext() {
  if (currentPage < data.total_pages) {
    currentPage++;
    loadImages();
  }
}

// Event Listeners
document.addEventListener("DOMContentLoaded", function () {
  createModal(); // Create modal on page load
  initializeDragAndDrop();
  loadImages();

  // Handle image clicks for preview
  document
    .getElementById("imageList")
    .addEventListener("click", function (event) {
      if (event.target.tagName === "IMG") {
        const fileId = event.target.dataset.fileId;
        if (fileId) {
          openModal(fileId);
        }
      }
    });

  // Handle download link clicks
  document
    .getElementById("imageList")
    .addEventListener("click", function (event) {
      if (event.target.classList.contains("download-link")) {
        event.preventDefault();
        const url = event.target.dataset.url;
        const filename = event.target.dataset.filename;
        downloadFile(url, filename);
      }

      if (event.target.classList.contains("share-link")) {
        const fileId = event.target.dataset.fileId;
        shareImage(fileId);
      }
    });

  // Handle file upload
  document
    .getElementById("images")
    .addEventListener("change", async function (e) {
      const files = e.target.files;
      if (files.length > 0) {
        await uploadFiles(files);
      }
    });
});

// Create modal HTML structure with iframe
function createModal() {
  const modal = document.createElement("div");
  modal.id = "imagePreviewModal";
  modal.className = "modal";
  modal.innerHTML = `
        <div class="modal-content">
            <span class="close-modal">&times;</span>
            <iframe id="modalPreview" src="" frameborder="0" allowfullscreen></iframe>
        </div>
    `;
  document.body.appendChild(modal);

  // Add modal close functionality
  const closeBtn = modal.querySelector(".close-modal");
  closeBtn.onclick = closeModal;
  modal.onclick = (e) => {
    if (e.target === modal) {
      closeModal();
    }
  };
}

// Modal control functions
function openModal(fileId) {
  const modal = document.getElementById("imagePreviewModal");
  const modalPreview = document.getElementById("modalPreview");
  // Use Google Drive's embed URL format
  modalPreview.src = `https://drive.google.com/file/d/${fileId}/preview`;
  modal.style.display = "block";

  // Add keyboard support for closing modal
  document.addEventListener("keydown", handleEscapeKey);
}

function closeModal() {
  const modal = document.getElementById("imagePreviewModal");
  const modalPreview = document.getElementById("modalPreview");
  modalPreview.src = ""; // Clear the iframe source
  modal.style.display = "none";
  document.removeEventListener("keydown", handleEscapeKey);
}

function handleEscapeKey(e) {
  if (e.key === "Escape") {
    closeModal();
  }
}

// Function to get the last like timestamp for a specific image
function getLastLikeTime(imageId) {
  const lastLikes = JSON.parse(localStorage.getItem("lastLikes")) || {};
  return lastLikes[imageId] || 0; // Default to 0 if no like timestamp is found
}

// Function to set the last like timestamp for a specific image
function setLastLikeTime(imageId, timestamp) {
  const lastLikes = JSON.parse(localStorage.getItem("lastLikes")) || {};
  lastLikes[imageId] = timestamp;
  localStorage.setItem("lastLikes", JSON.stringify(lastLikes));
}

async function toggleLike(imageId) {
  const currentTime = Date.now();
  const lastLikedTime = getLastLikeTime(imageId);

  // Check if less than 60 seconds have passed since the last like
  if (currentTime - lastLikedTime < 300) {
    return;
  }

  try {
    // Make API call to like the image
    const response = await fetch(getUrl("image/like"), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ image_id: imageId }),
    });

    if (response.ok) {
      const data = await response.json();
      const likeCountElement = document.getElementById(`like-count-${imageId}`);

      // Update like count
      likeCountElement.textContent = data.liked_count;

      // Create the heart pop effect
      const likeButton = likeCountElement.previousElementSibling; // Select the like button
      likeButton.style.transform = "scale(1.5)";

      setTimeout(() => {
        likeButton.style.transform = "";
      }, 500);

      // Store the current time as the last liked time for this image
      setLastLikeTime(imageId, currentTime);
    } else {
      showToast("Failed to like the image.");
    }
  } catch (error) {
    console.error("Error liking image:", error);
    showToast("An error occurred while liking the image.");
  }
}

// Function to handle sharing
function shareImage(fileId) {
  // Construct the Google Drive preview link
  const shareUrl = `https://drive.google.com/file/d/${fileId}/preview`;

  if (navigator.share) {
    // Use Web Share API if available
    navigator
      .share({
        title: "Check out this image!",
        url: shareUrl,
      })
      .then(() => console.log("Share successful!"))
      .catch((error) => console.log("Error sharing:", error));
  } else {
    // Fallback: Copy link to clipboard if Web Share API is not available
    navigator.clipboard.writeText(shareUrl).then(() => {
      showToast("Link copied to clipboard! You can paste it to share.");
    });
  }
}
