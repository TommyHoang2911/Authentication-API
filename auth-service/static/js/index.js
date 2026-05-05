const ui = {
    indicator: document.getElementById('status-indicator'),
    text: document.getElementById('status-text'),
    facebookButton: document.getElementById('facebook-login'),
    qrButton: document.getElementById('qr-test'),

    setPending() {
        this.indicator.innerHTML = `
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-yellow-400 opacity-75"></span>
            <span class="relative inline-flex rounded-full h-3 w-3 bg-yellow-500"></span>`;
        this.text.textContent = 'Đang kiểm tra hệ thống...';
        this.text.className = 'text-sm font-medium text-slate-300';
    },

    setOnline() {
        this.indicator.innerHTML = `
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
            <span class="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>`;
        this.text.textContent = 'Hệ thống Online (Database connected)';
        this.text.className = 'text-sm font-medium text-green-400';
    },

    setOffline() {
        this.indicator.innerHTML = `
            <span class="relative inline-flex rounded-full h-3 w-3 bg-red-500"></span>`;
        this.text.textContent = 'Hệ thống Offline hoặc Database mất kết nối';
        this.text.className = 'text-sm font-medium text-red-400';
    }
};

function showMessage(message) {
    window.alert(message);
}

function registerEventHandlers() {
    ui.facebookButton.addEventListener('click', () => {
        showMessage('Tính năng OAuth Facebook đang được cập nhật!');
    });

    ui.qrButton.addEventListener('click', () => {
        showMessage('Tính năng WebSocket QR Login đang được cập nhật trên giao diện!');
    });
}

async function checkReadiness() {
    ui.setPending();

    try {
        const response = await fetch('/health/readiness');
        if (response.ok) {
            ui.setOnline();
        } else {
            ui.setOffline();
        }
    } catch (error) {
        ui.setOffline();
    }
}

document.addEventListener('DOMContentLoaded', () => {
    registerEventHandlers();
    checkReadiness();
});
