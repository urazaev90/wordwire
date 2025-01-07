const clickSoundError = new Audio('/static/sounds/_click_error.mp3');

function playClickSoundError() {
    clickSoundError.pause();
    clickSoundError.currentTime = 0;
    clickSoundError.play().catch(error => console.error("Ошибка воспроизведения звука:", error));
}

document.addEventListener("DOMContentLoaded", function () {
    // Добавляем логику для открытия и закрытия модального окна
    const modal = document.getElementById('registration-modal');
    const overlay = document.getElementById('modal-overlay');
    const openModalButton = document.getElementById('open-modal-button'); // Изменили на кнопку с ID "open-modal-button"
    const closeModalButton = document.getElementById('close-modal');

    // Открытие модального окна по клику на кнопку
    openModalButton.addEventListener('click', function (event) {
        playClickSound();
        event.preventDefault(); // Отключаем стандартное действие кнопки
        modal.style.display = 'block';
        overlay.style.display = 'block';
        closeModalButton.style.display = 'block'; // Показываем крестик
    });

    // Закрытие модального окна по клику на крестик
    closeModalButton.addEventListener('click', function () {
        playClickSound();
        modal.style.display = 'none';
        overlay.style.display = 'none';
        closeModalButton.style.display = 'none'; // Скрываем крестик
    });

    // Закрытие модального окна по клику на затемненный фон
    overlay.addEventListener('click', function () {
        playClickSound();
        modal.style.display = 'none';
        overlay.style.display = 'none';
        closeModalButton.style.display = 'none'; // Скрываем крестик
    });
});

// Тут мы объединяем проверку паролей и отправку формы
async function validateAndSubmitForm(event) {
    // Останавливаем стандартное поведение отправки формы
    event.preventDefault();

    // Тут получаем значения пароля и подтверждения пароля
    const password = document.getElementById("password").value;
    const confirmPassword = document.getElementById("confirmPassword").value;

    // Тут находим элемент для вывода сообщения об ошибке
    const errorMessage = document.getElementById("error-message");

    // Проверяем, совпадают ли пароли
    if (password !== confirmPassword) {
        // Если пароли не совпадают, выводим сообщение и выходим из функции
        playClickSoundError();
        errorMessage.textContent = "Пароли не совпадают!";
        return; // Тут мы говорим "остановись, не отправляй данные"
    } else {
        // Если все нормально, очищаем сообщение об ошибке
        errorMessage.textContent = "";
    }

    // Оставновим выполнение кода дальше, если есть ошибка
    if (errorMessage.textContent) {
        return;
    }


    // отправка формы на сервер джаваскриптом через formdata, а если есть ошибки сервер нам шлет их через json
    // Собираем данные из формы
    const form = event.target;
    const formData = new FormData(form);

    try {
        // Тут мы отправляем запрос на сервер
        const response = await fetch(form.action, {
            method: form.method,
            body: formData, // Передаем данные из формы
        });

        // Проверяем успешность ответа от сервера
        if (response.ok) {
            // Получаем результат в формате JSON
            const result = await response.json();

            // Проверяем, есть ли ошибка в результате
            if (result.error) {
                // Если ошибка есть, показываем сообщение
                playClickSoundError();
                errorMessage.textContent = result.error;
                reloadCaptcha();
            } else {
                // Если ошибки нет, перенаправляем на страницу профиля
                window.location.href = "/setting";
            }
        } else {
            // Если сервер вернул ошибку, показываем сообщение
            playClickSoundError();
            errorMessage.textContent = "Ошибка на сервере. Попробуйте позже.";
            reloadCaptcha();
        }
    } catch (error) {
        // Тут мы обрабатываем ошибки соединения
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