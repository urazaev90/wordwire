document.addEventListener("DOMContentLoaded", () => {
    const showCaptcha = document.getElementById("showCaptcha").value === "true";
    const captchaContainer = document.getElementById("captcha-container");

    if (showCaptcha) {
        captchaContainer.style.display = "block";
        loadCaptcha();
    }
});

async function validateLoginForm(event) {
    event.preventDefault();
    const form = event.target;
    const formData = new FormData(form);

    const errorMessage = document.getElementById("error-message");
    try {
        const response = await fetch(form.action, {
            method: form.method,
            body: formData,
        });

        const result = await response.json();
        if (result.error) {
            errorMessage.textContent = result.error;

            // Показываем капчу, если сервер сообщил, что она нужна
            if (result.showCaptcha === "true") {

                document.getElementById("captcha-container").style.display = "block";
                reloadCaptcha();
            }
        } else {
            window.location.href = "/teaching";
        }
    } catch (error) {
        console.error("Ошибка:", error);
        errorMessage.textContent = "Ошибка соединения. Попробуйте позже.";
    }
}

async function loadCaptcha() {
    try {
        const response = await fetch("/generate-captcha");
        const data = await response.json();
        document.getElementById("captchaID").value = data.captchaID;
        document.getElementById("captcha-image").src = data.captchaURL;
    } catch (error) {
        console.error("Ошибка загрузки капчи:", error);
    }
}

function reloadCaptcha() {
    loadCaptcha();
}