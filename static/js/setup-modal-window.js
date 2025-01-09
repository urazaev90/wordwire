document.addEventListener('DOMContentLoaded', function () {
    setupModal('logout-link', 'logout-modal', 'close');
    setupModal('thankyou', 'thankyou-modal', 'close');

    document.getElementById('logout-link').addEventListener('click', function () {
        fetch('/api/get_user_login')
            .then(response => response.json())
            .then(data => {
                document.getElementById('user-login').innerText = data.login;
            });
    });

    document.getElementById('cancel-logout').addEventListener('click', function () {
        playClickSound();
        document.getElementById('logout-modal').style.display = 'none';
    });

    document.getElementById('confirm-logout').addEventListener('click', function () {
        playClickSound();
        fetch('/logout', { method: 'POST' })
            .then(() => {
                window.location.href = '/';
            });
    });
});