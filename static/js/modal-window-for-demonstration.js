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