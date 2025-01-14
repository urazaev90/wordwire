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