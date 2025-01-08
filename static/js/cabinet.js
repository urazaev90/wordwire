const clickSound = new Audio('/static/sounds/_click_sound.mp3');

function playClickSound() {
    clickSound.pause();
    clickSound.currentTime = 0;
    clickSound.play().catch(error => console.error("Ошибка воспроизведения звука:", error));
}

window.addEventListener("load", () => {
    playClickSound();
});

document.addEventListener('DOMContentLoaded', function () {

    document.getElementById('logout-link').addEventListener('click', function () {
        playClickSound();
        fetch('/api/get_user_login')
            .then(response => response.json())
            .then(data => {
                document.getElementById('user-login').innerText = data.login;
                document.getElementById('logout-modal').style.display = 'block';
            });
    });

    document.querySelector('.close').addEventListener('click', function () {
        document.getElementById('logout-modal').style.display = 'none';
        playClickSound();
    });

    document.getElementById('cancel-logout').addEventListener('click', function () {
        playClickSound();
        document.getElementById('logout-modal').style.display = 'none';
    });

    document.getElementById('confirm-logout').addEventListener('click', function () {
        playClickSound();
        fetch('/logout', {method: 'POST'})
            .then(() => {
                window.location.href = '/';
            });
    });

    window.onclick = function (event) {
        if (event.target == document.getElementById('logout-modal')) {
            playClickSound();
            document.getElementById('logout-modal').style.display = 'none';
        }
    };
});