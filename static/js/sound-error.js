const clickSoundError = new Audio('/static/sounds/_click_error.mp3');

function playClickSoundError() {
    clickSoundError.pause();
    clickSoundError.currentTime = 0;
    clickSoundError.play().catch(error => console.error("Ошибка воспроизведения звука:", error));
}