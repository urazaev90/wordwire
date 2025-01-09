function setupModal(triggerId, modalId, closeClass) {
    const trigger = document.getElementById(triggerId);
    const modal = document.getElementById(modalId);
    const closeBtn = modal.querySelector(`.${closeClass}`);

    trigger.addEventListener('click', function () {
        playClickSound();
        modal.style.display = 'block';
    });

    closeBtn.addEventListener('click', function () {
        playClickSound();
        modal.style.display = 'none';
    });

    window.addEventListener('click', function (event) {
        if (event.target === modal) {
            playClickSound();
            modal.style.display = 'none';
        }
    });
}