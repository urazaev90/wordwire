async function validateAndSubmitForm(event) {
    event.preventDefault();

    const password = document.getElementById("password").value;
    const confirmPassword = document.getElementById("confirmPassword").value;
    const errorMessage = document.getElementById("error-message");
    const form = event.target;
    const formData = new FormData(form);

    if (password !== confirmPassword) {
        errorMessage.textContent = "Пароли не совпадают!";
        return;
    } else {
        errorMessage.textContent = "";
    }

    if (errorMessage.textContent) {
        return;
    }

    try {
        const response = await fetch(form.action, {
            method: form.method,
            body: formData,
        });

        if (response.ok) {
            const result = await response.json();

            if (result.error) {
                errorMessage.textContent = result.error;
                reloadCaptcha();
            } else {
                window.location.href = "/teaching";
            }
        } else {
            errorMessage.textContent = "Ошибка на сервере. Попробуйте позже.";
            reloadCaptcha();
        }
    } catch (error) {
        console.error("Ошибка:", error);
        errorMessage.textContent = "Ошибка. Проверьте подключение к интернету.";
        reloadCaptcha();
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

document.addEventListener("DOMContentLoaded", loadCaptcha);