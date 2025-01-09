const clickSound = new Audio('/static/sounds/_click_sound.mp3');

function playClickSound() {
    clickSound.pause();
    clickSound.currentTime = 0;
    clickSound.play().catch(error => console.error("Ошибка воспроизведения звука:", error));
}

window.addEventListener("load", () => {
    playClickSound();
});