// Отключение контекстного меню
document.addEventListener('contextmenu', (event) => {
    event.preventDefault();
});

// Отключение длительных нажатий на мобильных устройствах
document.addEventListener('touchstart', (event) => {
    if (event.touches.length > 1) { // Если больше одного пальца, предотвращаем
        event.preventDefault();
    }
}, { passive: false });

// Отключение контекстного меню на интерактивных элементах
const elements = document.querySelectorAll('a, img');
elements.forEach((element) => {
    element.addEventListener('touchstart', (event) => {
        event.preventDefault(); // Отключение действия при долгом нажатии
    }, { passive: false });

    element.addEventListener('touchend', (event) => {
        event.preventDefault(); // Убедиться, что нет последствий нажатия
    }, { passive: false });

    element.addEventListener('contextmenu', (event) => {
        event.preventDefault(); // Запасной вариант для отключения меню
    });
});

// Отключение выделения текста
document.addEventListener('selectstart', (event) => {
    event.preventDefault();
});

// Отключение дополнительных действий, таких как масштабирование
document.addEventListener('gesturestart', (event) => {
    event.preventDefault();
});
