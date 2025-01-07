document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll('input[type="checkbox"]').forEach((checkbox) => {
        checkbox.addEventListener('change', function (event) {
            const wordId = event.target.name;
            const label = event.target.checked ? 2 : 1;

            fetch(`/selected`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8'
                },
                body: `id=${wordId}&label=${label}`
            })
                .then(response => {
                    if (!response.ok) {
                        console.error('Не удалось обновить метку слова:', response.statusText);
                    }
                    location.reload();
                })
                .catch(error => console.error('Ошибка:', error));
        });
    });

    document.querySelectorAll('[data-archive]').forEach((link) => {
        link.addEventListener('click', function (event) {
            event.preventDefault();
            const wordId = this.getAttribute('data-archive');

            fetch(`/add_to_archive`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8'
                },
                body: `archive_word_id=${wordId}`
            })
                .then(response => {
                    if (!response.ok) {
                        console.error('Не удалось архивировать слово:', response.statusText);
                    }
                    location.reload();
                })
                .catch(error => console.error('Ошибка:', error));
        });
    });
});