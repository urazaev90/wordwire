document.addEventListener('DOMContentLoaded', function () {

    document.getElementById('thankyou').addEventListener('click', function () {
        playClickSound();
        document.getElementById('thankyou-modal').style.display = 'block';

    });

    document.getElementById('thankyou').addEventListener('click', function () {
        playClickSound();
    });

    document.querySelector('.thankyou .close').addEventListener('click', function () {
        document.getElementById('thankyou-modal').style.display = 'none';
        playClickSound();
    });

    window.addEventListener('click', function (event) {
        if (event.target === document.getElementById('thankyou-modal')) {
            playClickSound();
            document.getElementById('thankyou-modal').style.display = 'none';
        }
    });
});