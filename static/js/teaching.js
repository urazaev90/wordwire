// Создаем глобальный объект для звука клика
const clickSound = new Audio('/static/sounds/_click_sound.mp3');

// Функция для воспроизведения звука с предотвращением многократного запуска
function playClickSound() {
    clickSound.pause(); // Останавливаем звук, если он уже воспроизводится
    clickSound.currentTime = 0; // Сбрасываем звук в начало
    clickSound.play().catch(error => console.error("Ошибка воспроизведения звука:", error));
}

window.addEventListener("load", () => {
    playClickSound(); // Воспроизведение звука при загрузке страницы
});


document.addEventListener("DOMContentLoaded", function () {
    let words = [];
    let currentIndex = 0;
    let history = [];

    async function fetchWords() {
        try {
            document.getElementById("loading").style.display = "block";
            document.getElementById("current-word").style.display = "none";

            const response = await fetch('/api/words');
            if (!response.ok) {
                throw new Error(`Ошибка загрузки слов: ${response.statusText}`);
            }

            words = await response.json();
            document.getElementById("loading").style.display = "none";
            document.getElementById("current-word").style.display = "block";

            shuffle(words);
            document.getElementById("words-left").textContent = words.length - currentIndex;
            showWord();
        } catch (error) {
            console.error("Ошибка:", error);
            document.getElementById("loading").style.display = "none";
            document.getElementById("current-word").style.display = "block";
            document.getElementById("current-word").textContent = "Не выбрано ни одного слова.";
        }
    }

    function shuffle(array) {
        for (let i = array.length - 1; i > 0; i--) {
            const j = Math.floor(Math.random() * (i + 1));
            [array[i], array[j]] = [array[j], array[i]];
        }
    }

    // Функция для отображения текущего слова и изменения состояния иконок
    function showWord() {
        if (words.length === 0) {
            document.getElementById("current-word").textContent = "Не выбрано ни одного слова.";
            return;
        }

        const backIcon = document.getElementById("back-icon");
        const forwardIcon = document.getElementById("forward-icon");

        if (currentIndex < words.length) {
            // Проверка и смена иконки для кнопки "Назад"
            backIcon.src = currentIndex > 0 ? '/static/images/left.png' : '/static/images/left_not_active.png';
            backIcon.className = currentIndex > 0 ? 'button' : 'not-active';

            // Проверка и смена иконки для кнопки "Вперед"
            forwardIcon.src = currentIndex < words.length - 1 ? '/static/images/right.png' : '/static/images/right_not_active.png';
            forwardIcon.className = currentIndex < words.length - 1 ? 'button' : 'not-active';
        }

        // Обновление текущего слова на экране
        if (currentIndex >= 0 && currentIndex < words.length) {
            const word = words[currentIndex];
            document.getElementById("current-word").textContent = word.word;
            document.getElementById("translation").style.visibility = "hidden";
            document.getElementById("words-left").textContent = words.length - currentIndex;
        }
    }

    // Обработчик события для кнопки "Вперед"
    document.getElementById("forward-icon").addEventListener("click", () => {
        if (currentIndex < words.length - 1) {
            history.push(currentIndex);
            currentIndex++;
            showWord();
            playClickSound(); // Воспроизведение звука при клике на кнопку "Вперед"
        }
    });

    // Обработчик события для кнопки "Назад"
    document.getElementById("back-icon").addEventListener("click", () => {
        if (history.length > 0) {
            currentIndex = history.pop();
            showWord();
            playClickSound(); // Воспроизведение звука при клике на кнопку "Назад"
        }
    });

    document.getElementById("restart-icon").addEventListener("click", () => {
        location.reload();
    });

    document.getElementById("translate-icon").addEventListener("click", () => {
        const translationElement = document.getElementById("translation");

        playClickSound(); // Воспроизведение звука при клике на кнопку

        if (translationElement.style.visibility === "hidden") {
            const word = words[currentIndex];
            translationElement.innerHTML = `
                        <p id="transcription">${word.transcription}</p>
                        <p id="translation-text">${word.translation}</p>
                    `;
            translationElement.style.visibility = "visible";
        } else {
            translationElement.style.visibility = "hidden";
        }
    });

    function playSound(word) {
        const audio = new Audio(`/static/sounds/${word}.mp3`);
        audio.play().catch(error => console.error("Ошибка воспроизведения звука:", error));
    }

    document.getElementById("sound-icon").addEventListener("click", () => {
        if (currentIndex >= 0 && currentIndex < words.length) {
            playSound(words[currentIndex].word);
        }
    });

    fetchWords();
});