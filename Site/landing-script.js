function showNameInput() {
  document.getElementById("nameModal").style.display = "block";
}

function closeNameModal() {
  document.getElementById("nameModal").style.display = "none";
}

function saveNameAndRedirect() {
  const nameInput = document.getElementById("nameInput");
  const name = nameInput.value.trim();

  if (name !== "") {
    localStorage.setItem("userName", name);

    const successMessageContainer =
      document.getElementsByClassName("name-modal-content")[0];
    successMessageContainer.innerHTML = ""; // Clear previous messages

    const successMessage = document.createElement("div");
    successMessage.classList.add("success-message");
    successMessage.textContent = `Welcome, ${name} 🙏`;

    successMessageContainer.appendChild(successMessage);

    setTimeout(() => {
      window.location.href = "home.html";
    }, 2000);
  } else {
    alert("Please enter your name.");
  }
}
