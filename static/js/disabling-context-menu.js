// Отключение контекстного меню на странице
document.addEventListener('contextmenu', (event) => {
    event.preventDefault();
});

// Отключение длительных нажатий на мобильных устройствах
document.addEventListener('touchstart', (event) => {
    if (event.touches.length > 1) { // Если больше одного пальца, предотвращаем
        event.preventDefault();
    }
});

// Отключение выделения текста (актуально для iOS)
document.addEventListener('selectstart', (event) => {
    event.preventDefault();
});

// Отключение всплывающего меню на iOS при длительном нажатии
document.addEventListener('touchend', (event) => {
    if (event.touches.length === 1) { // Только один палец
        event.preventDefault();
    }
}, { passive: false });

// Отключение дополнительных действий (например, масштабирование)
document.addEventListener('gesturestart', (event) => {
    event.preventDefault();
});
