// ============================================
// Configuration
// ============================================
const CONFIG = {
    API: {
        VERIFY_QR: '/verify_qr',
        CANCEL_QR: '/cancel_qr'
    },
    UI: {
        MAIN_VIEW: 'main-view',
        SCANNER_VIEW: 'scanner-view',
        CONFIRM_MODAL: 'confirm-modal',
        TOKEN_INPUT: 'device-a-token',
        MANUAL_CODE_INPUT: 'manual-code',
        READER: 'reader'
    },
    SCANNER: {
        FPS: 10,
        QRBOX_SIZE: 250
    },
    MESSAGES: {
        INVALID_QR: 'Invalid QR Code format!',
        ENTER_CODE: 'Please enter a code!',
        ENTER_TOKEN: 'Vui lòng nhập Access Token của Device A trước!',
        VERIFYING: 'Verifying...',
        SUCCESS: 'Success! Device B is now logged in.',
        VERIFY_FAILED: 'Failed to verify: ',
        NETWORK_ERROR: 'Network error occurred.',
        DENIED: 'Login request denied.'
    }
};

// ============================================
// UI Manager - Handles DOM manipulation and UI state
// ============================================
const uiManager = {
    elements: {},

    init() {
        this.elements = {
            mainView: document.getElementById(CONFIG.UI.MAIN_VIEW),
            scannerView: document.getElementById(CONFIG.UI.SCANNER_VIEW),
            confirmModal: document.getElementById(CONFIG.UI.CONFIRM_MODAL),
            tokenInput: document.getElementById(CONFIG.UI.TOKEN_INPUT),
            manualCodeInput: document.getElementById(CONFIG.UI.MANUAL_CODE_INPUT),
            reader: document.getElementById(CONFIG.UI.READER)
        };
    },

    showScanner() {
        this.elements.mainView.classList.add('hidden');
        this.elements.scannerView.classList.remove('hidden');
    },

    hideScanner() {
        this.elements.scannerView.classList.add('hidden');
        this.elements.mainView.classList.remove('hidden');
    },

    showConfirmModal() {
        this.elements.confirmModal.classList.remove('hidden');
    },

    hideConfirmModal() {
        this.elements.confirmModal.classList.add('hidden');
    },

    resetUI() {
        this.elements.manualCodeInput.value = '';
        this.hideConfirmModal();
        this.elements.mainView.classList.remove('hidden');
        this._resetConfirmButton();
    },

    _resetConfirmButton() {
        const allowBtn = this.elements.confirmModal.querySelectorAll('button')[1];
        allowBtn.innerText = 'Allow';
        allowBtn.disabled = false;
    },

    setConfirmButtonLoading() {
        const allowBtn = this.elements.confirmModal.querySelectorAll('button')[1];
        allowBtn.innerText = CONFIG.MESSAGES.VERIFYING;
        allowBtn.disabled = true;
    },

    getToken() {
        return this.elements.tokenInput.value.trim();
    },

    getManualCode() {
        return this.elements.manualCodeInput.value.trim();
    },

    showAlert(message) {
        window.alert(message);
    }
};

// ============================================
// Scanner Manager - Handles QR scanning
// ============================================
const scannerManager = {
    scanner: null,

    init() {
        this.scanner = new Html5QrcodeScanner(CONFIG.UI.READER, {
            fps: CONFIG.SCANNER.FPS,
            qrbox: {
                width: CONFIG.SCANNER.QRBOX_SIZE,
                height: CONFIG.SCANNER.QRBOX_SIZE
            },
            aspectRatio: 1.0
        }, false);
    },

    start(onSuccess, onFailure) {
        this.scanner.render(onSuccess, onFailure);
    },

    stop() {
        if (this.scanner) {
            this.scanner.clear().catch(error => console.error('Failed to clear scanner.', error));
        }
    }
};

// ============================================
// API Manager - Handles API calls
// ============================================
const apiManager = {
    async verifyQR(token, qrCode) {
        const response = await fetch(CONFIG.API.VERIFY_QR, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ code: qrCode })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || 'Unknown error');
        }

        return response;
    },

    async cancelQR(token, qrCode) {
        return fetch(CONFIG.API.CANCEL_QR, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ code: qrCode })
        });
    }
};

// ============================================
// Auth Manager - Handles authentication flow
// ============================================
const authManager = {
    scannedQrId: null,

    setScannedQrId(qrId) {
        this.scannedQrId = qrId;
    },

    getScannedQrId() {
        return this.scannedQrId;
    },

    clearScannedQrId() {
        this.scannedQrId = null;
    },

    parseQRCode(decodedText) {
        try {
            const data = JSON.parse(decodedText);
            return data.code || data.id;
        } catch (e) {
            return decodedText;
        }
    },

    validateToken() {
        const token = uiManager.getToken();
        if (!token) {
            uiManager.showAlert(CONFIG.MESSAGES.ENTER_TOKEN);
            return false;
        }
        return token;
    },

    async confirmLogin() {
        const token = this.validateToken();
        if (!token) return;

        uiManager.setConfirmButtonLoading();

        try {
            await apiManager.verifyQR(token, this.scannedQrId);
            uiManager.showAlert(CONFIG.MESSAGES.SUCCESS);
            this._resetFlow();
        } catch (error) {
            uiManager.showAlert(`${CONFIG.MESSAGES.VERIFY_FAILED}${error.message}`);
            this._resetFlow();
        }
    },

    async cancelLogin() {
        const token = this.validateToken();
        if (!token) return;

        try {
            await apiManager.cancelQR(token, this.scannedQrId);
            uiManager.showAlert(CONFIG.MESSAGES.DENIED);
        } catch (error) {
            console.error('Cancel QR error:', error);
        }
        this._resetFlow();
    },

    _resetFlow() {
        this.clearScannedQrId();
        uiManager.resetUI();
    }
};

// ============================================
// Event Handlers
// ============================================
function openScanner() {
    uiManager.showScanner();
    scannerManager.start(onScanSuccess, onScanFailure);
}

function closeScanner() {
    scannerManager.stop();
    uiManager.hideScanner();
}

function onScanSuccess(decodedText, decodedResult) {
    console.log(`Scan result: ${decodedText}`);
    closeScanner();

    const qrId = authManager.parseQRCode(decodedText);
    if (qrId) {
        authManager.setScannedQrId(qrId);
        uiManager.showConfirmModal();
    } else {
        uiManager.showAlert(CONFIG.MESSAGES.INVALID_QR);
    }
}

function onScanFailure(error) {
    // Silent failure for continuous scanning
}

function handleManualInput() {
    const code = uiManager.getManualCode();
    if (!code) {
        uiManager.showAlert(CONFIG.MESSAGES.ENTER_CODE);
        return;
    }

    authManager.setScannedQrId(code);
    uiManager.showConfirmModal();
}

function confirmLogin() {
    authManager.confirmLogin();
}

function cancelLogin() {
    authManager.cancelLogin();
}

function closeModal() {
    uiManager.resetUI();
}

function handleOverlayClick(event) {
    // Kiểm tra xem người dùng click chính xác vào cái nền đen (có id="confirm-modal")
    // chứ không phải click vào vùng trắng bên trong
    if (event.target.id === 'confirm-modal') {
        closeModal();
    }
}

// ============================================
// Initialization
// ============================================
document.addEventListener('DOMContentLoaded', () => {
    uiManager.init();
    scannerManager.init();
});