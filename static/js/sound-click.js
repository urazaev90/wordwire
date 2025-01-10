const soundClick = new Audio('/static/sounds/_click_sound.mp3');

function playClickSound() {
    soundClick.pause();
    soundClick.currentTime = 0;
    soundClick.play().catch(error => console.error("Ошибка воспроизведения звука:", error));
}