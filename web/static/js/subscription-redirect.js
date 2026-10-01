// Shared by the subscription success and cancel pages. Redirects to the main
// app after the delay given in the script tag's data-redirect-delay attribute,
// and wires up the cancel page's "Try Subscribing Again" link.
(function () {
    const delay = Number(document.currentScript.dataset.redirectDelay) || 5000;

    const tryAgainBtn = document.getElementById('try-again-btn');
    if (tryAgainBtn) {
        tryAgainBtn.addEventListener('click', (e) => {
            e.preventDefault();
            window.location.href = '/';
        });
    }

    setTimeout(() => {
        window.location.href = '/';
    }, delay);
})();
