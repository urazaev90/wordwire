const clickSoundError = new Audio('/static/sounds/_click_error.mp3');

function playClickSoundError() {
    clickSoundError.pause();
    clickSoundError.currentTime = 0;
    clickSoundError.play().catch(error => console.error("Ошибка воспроизведения звука:", error));
}

document.addEventListener("DOMContentLoaded", function () {
    const modal = document.getElementById('registration-modal');
    const overlay = document.getElementById('modal-overlay');
    const openModalButton = document.getElementById('open-modal-button');
    const closeModalButton = document.getElementById('close-modal');

    openModalButton.addEventListener('click', function (event) {
        playClickSound();
        event.preventDefault();
        modal.style.display = 'block';
        overlay.style.display = 'block';
        closeModalButton.style.display = 'block';
    });

    closeModalButton.addEventListener('click', function () {
        playClickSound();
        modal.style.display = 'none';
        overlay.style.display = 'none';
        closeModalButton.style.display = 'none';
    });

    overlay.addEventListener('click', function () {
        playClickSound();
        modal.style.display = 'none';
        overlay.style.display = 'none';
        closeModalButton.style.display = 'none';
    });
});

async function validateAndSubmitForm(event) {
    event.preventDefault();

    const password = document.getElementById("password").value;
    const confirmPassword = document.getElementById("confirmPassword").value;
    const errorMessage = document.getElementById("error-message");

    if (password !== confirmPassword) {
        playClickSoundError();
        errorMessage.textContent = "Пароли не совпадают!";
        return;
    } else {
        errorMessage.textContent = "";
    }

    if (errorMessage.textContent) {
        return;
    }

    const form = event.target;
    const formData = new FormData(form);

    try {
        const response = await fetch(form.action, {
            method: form.method,
            body: formData,
        });

        if (response.ok) {
            const result = await response.json();

            if (result.error) {
                playClickSoundError();
                errorMessage.textContent = result.error;
                reloadCaptcha();
            } else {
                window.location.href = "/setting";
            }
        } else {
            playClickSoundError();
            errorMessage.textContent = "Ошибка на сервере. Попробуйте позже.";
            reloadCaptcha();
        }
    } catch (error) {
        console.error("Ошибка:", error);
        playClickSoundError();
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