// ============================================
// Constants
// ============================================
const CONFIG = {
    DEVICE_ID: 'device-webpage-test',
    QR_CODE: {
        SIZE: 200,
        COLOR_DARK: '#000000',
        COLOR_LIGHT: '#ffffff'
    },
    API: {
        GENERATE_QR: '/generate_qr',
        EXCHANGE_CODE: '/exchange_code',
        HEALTH_CHECK: '/health/readiness'
    },
    UI: {
        TERMINAL_ID: 'qr-terminal',
        QR_CONTAINER_ID: 'qr-image-container',
        STATUS_INDICATOR_ID: 'status-indicator',
        STATUS_TEXT_ID: 'status-text',
        FACEBOOK_BUTTON_ID: 'facebook-login'
    }
};

// ============================================
// Status Manager - Handles system status UI
// ============================================
const statusManager = {
    indicator: null,
    text: null,

    init() {
        this.indicator = document.getElementById(CONFIG.UI.STATUS_INDICATOR_ID);
        this.text = document.getElementById(CONFIG.UI.STATUS_TEXT_ID);
    },

    _setStatus(message, indicatorHTML, textClass) {
        this.indicator.innerHTML = indicatorHTML;
        this.text.textContent = message;
        this.text.className = textClass;
    },

    setPending() {
        this._setStatus(
            'Đang kiểm tra hệ thống...',
            `<span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-yellow-400 opacity-75"></span>
             <span class="relative inline-flex rounded-full h-3 w-3 bg-yellow-500"></span>`,
            'text-sm font-medium text-slate-300'
        );
    },

    setOnline() {
        this._setStatus(
            'Hệ thống Online (Database connected)',
            `<span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
             <span class="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>`,
            'text-sm font-medium text-green-400'
        );
    },

    setOffline() {
        this._setStatus(
            'Hệ thống Offline hoặc Database mất kết nối',
            `<span class="relative inline-flex rounded-full h-3 w-3 bg-red-500"></span>`,
            'text-sm font-medium text-red-400'
        );
    }
};

// ============================================
// Health Check Manager
// ============================================
const healthCheckManager = {
    async checkReadiness() {
        statusManager.setPending();
        try {
            const response = await fetch(CONFIG.API.HEALTH_CHECK);
            statusManager[response.ok ? 'setOnline' : 'setOffline']();
        } catch (error) {
            console.error('Health check error:', error);
            statusManager.setOffline();
        }
    }
};

// ============================================
// Terminal Logger - For QR flow debugging
// ============================================
const terminalLogger = {
    terminal: null,

    init() {
        this.terminal = document.getElementById(CONFIG.UI.TERMINAL_ID);
    },

    show() {
        this.terminal.classList.remove('hidden');
    },

    clear() {
        this.terminal.innerText = '';
    },

    log(tag, message) {
        const timestamp = `[${tag}]`;
        this.terminal.innerText += `${timestamp} ${message}\n`;
    },

    logSystem(message) {
        this.log('System', message);
    },

    logAPI(message) {
        this.log('API', message);
    },

    logSuccess(message) {
        this.log('Success', message);
    },

    logError(message) {
        this.log('Error', message);
    },

    logWebSocket(message) {
        this.log('WebSocket', message);
    },

    logException(message) {
        this.log('Exception', message);
    },

    logInstructions() {
        this.terminal.innerText += `========================================\n`;
        this.terminal.innerText += `HƯỚNG DẪN TEST (DEVICE A):\n`;
        this.terminal.innerText += `Hãy dùng Postman gọi POST /verify_qr với body chứa QR ID này và gửi kèm Bearer Token của một User đã đăng nhập.\n`;
        this.terminal.innerText += `========================================\n\n`;
    }
};

// ============================================
// QR Code Manager - Renders QR codes
// ============================================
const qrCodeManager = {
    container: null,

    init() {
        this.container = document.getElementById(CONFIG.UI.QR_CONTAINER_ID);
    },

    render(qrCode) {
        this.container.classList.remove('hidden');
        this.container.innerHTML = '';

        new QRCode(this.container, {
            text: qrCode,
            width: CONFIG.QR_CODE.SIZE,
            height: CONFIG.QR_CODE.SIZE,
            colorDark: CONFIG.QR_CODE.COLOR_DARK,
            colorLight: CONFIG.QR_CODE.COLOR_LIGHT,
            correctLevel: QRCode.CorrectLevel.H
        });
    }
};

// ============================================
// WebSocket Manager - Handles WS communication
// ============================================
const webSocketManager = {
    ws: null,

    getProtocol() {
        return window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    },

    getUrl(qrCode) {
        const protocol = this.getProtocol();
        return `${protocol}//${window.location.host}/ws?code=${qrCode}`;
    },

    connect(qrCode, onMessage, onError, onClose) {
        const url = this.getUrl(qrCode);
        terminalLogger.logWebSocket(`Đang kết nối tới ${url}...`);

        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
            terminalLogger.logWebSocket('Đã kết nối! Đang chờ Device A quét mã...');
        };

        this.ws.onmessage = (event) => {
            terminalLogger.logWebSocket(`Nhận được tin nhắn: ${event.data}`);
            onMessage(event.data);
        };

        this.ws.onerror = (error) => {
            terminalLogger.logWebSocket('Đã xảy ra lỗi kết nối.');
            if (onError) onError(error);
        };

        this.ws.onclose = () => {
            terminalLogger.logWebSocket('Kết nối đã đóng.');
            if (onClose) onClose();
        };
    },

    close() {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.close();
        }
    }
};

// ============================================
// API Manager - Handles API calls
// ============================================
const apiManager = {
    async generateQR(deviceId) {
        terminalLogger.logAPI(`Gọi POST ${CONFIG.API.GENERATE_QR}...`);

        const response = await fetch(CONFIG.API.GENERATE_QR, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ device_id: deviceId })
        });

        if (!response.ok) {
            throw new Error('Server không thể tạo mã QR');
        }

        return response.json();
    },

    async exchangeCode(tempCode) {
        terminalLogger.logAPI('Đang dùng temp_code để lấy JWT Token...');

        const response = await fetch(CONFIG.API.EXCHANGE_CODE, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ temp_code: tempCode })
        });

        if (!response.ok) {
            throw new Error('Lỗi khi exchange code!');
        }

        return response.json();
    }
};

// ============================================
// QR Flow Manager - Orchestrates QR login
// ============================================
const qrFlowManager = {
    async start() {
        terminalLogger.init();
        terminalLogger.show();
        terminalLogger.clear();
        terminalLogger.logSystem('Bắt đầu luồng đăng nhập QR...');

        qrCodeManager.init();

        try {
            // Step 1: Generate QR code
            const genData = await apiManager.generateQR(CONFIG.DEVICE_ID);
            const qrId = genData.code;

            terminalLogger.logSuccess(`Device B đã tạo QR thành công!\nQR code: ${qrId}`);
            terminalLogger.logInstructions();

            // Step 2: Render QR code
            const qrData = JSON.stringify({ code: qrId });
            qrCodeManager.render(qrData);

            // Step 3: Connect WebSocket and wait for verification
            this._setupWebSocket(qrId);

        } catch (error) {
            terminalLogger.logException(error.message);
        }
    },

    _setupWebSocket(qrId) {
        webSocketManager.connect(
            qrId,
            (data) => this._handleWebSocketMessage(data),
            () => { /* error handler */ },
            () => { /* close handler */ }
        );
    },

    async _handleWebSocketMessage(messageData) {
        try {
            const wsMsg = JSON.parse(messageData);

            if (wsMsg.temp_code) {
                const tokenData = await apiManager.exchangeCode(wsMsg.temp_code);
                terminalLogger.logSuccess(`Đăng nhập thành công! 🎉`);
                terminalLogger.logSuccess(`Access Token: ${tokenData.token}...`);
                webSocketManager.close();
            }
        } catch (error) {
            terminalLogger.logError(error.message);
        }
    }
};

// ============================================
// Event Handler Manager
// ============================================
const eventManager = {
    registerEventHandlers() {
        const facebookButton = document.getElementById(CONFIG.UI.FACEBOOK_BUTTON_ID);
        if (facebookButton) {
            facebookButton.addEventListener('click', () => {
                this._showMessage('Tính năng OAuth Facebook đang được cập nhật!');
            });
        }
    },

    _showMessage(message) {
        window.alert(message);
    }
};

// ============================================
// Initialization
// ============================================
document.addEventListener('DOMContentLoaded', () => {
    statusManager.init();
    eventManager.registerEventHandlers();
    healthCheckManager.checkReadiness();
});