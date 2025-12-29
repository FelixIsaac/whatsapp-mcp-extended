class PairingManager {
    constructor() {
        const currentProtocol = window.location.protocol;
        const currentHost = window.location.hostname;
        this.apiBaseUrl = `${currentProtocol}//${currentHost}:8180/api`;
        // Use stored API key or default test key (avoid prompt on page load)
        this.apiKey = localStorage.getItem('api_key') || 'test_key_for_build_verification_only';
        this.pollingInterval = null;
    }

    async generateCode() {
        const phoneInput = document.getElementById('phone-input');
        const phoneError = document.getElementById('phone-error');
        const phoneNumber = phoneInput.value.trim();

        // Remove leading + if present (API expects just digits)
        const cleanedPhone = phoneNumber.replace(/^\+/, '');

        if (!cleanedPhone) {
            phoneError.textContent = 'Phone number is required';
            return;
        }

        phoneError.textContent = '';

        try {
            const response = await fetch(`${this.apiBaseUrl}/pair`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-API-Key': this.apiKey
                },
                body: JSON.stringify({ phone_number: cleanedPhone })
            });

            const data = await response.json();

            if (!data.success) {
                phoneError.textContent = data.error || 'Failed to generate code';
                // If auth error, prompt for API key
                if (data.error && data.error.includes('Unauthorized')) {
                    this.promptForApiKey();
                }
                return;
            }

            // Show code section
            document.getElementById('phone-section').classList.add('hidden');
            document.getElementById('code-section').classList.remove('hidden');
            document.getElementById('code-text').textContent = data.code;

            // Start countdown and polling
            this.startCountdown(data.expires_in);
            this.startPolling();

        } catch (error) {
            phoneError.textContent = `Error: ${error.message}`;
            console.error('Pairing error:', error);
        }
    }

    startCountdown(seconds) {
        const countdownEl = document.getElementById('countdown-text');
        let remaining = seconds;

        const interval = setInterval(() => {
            remaining--;
            countdownEl.textContent = `${remaining}s`;

            if (remaining <= 0) {
                clearInterval(interval);
                countdownEl.textContent = 'Expired';
                document.getElementById('status-text').textContent = 'Code expired. Please try again.';
            }
        }, 1000);
    }

    async startPolling() {
        this.pollingInterval = setInterval(async () => {
            try {
                const response = await fetch(`${this.apiBaseUrl}/pairing`, {
                    headers: { 'X-API-Key': this.apiKey }
                });
                const data = await response.json();

                if (data.complete) {
                    this.stopPolling();
                    this.showDashboard();
                } else if (data.error) {
                    this.stopPolling();
                    document.getElementById('status-text').innerHTML =
                        `<i class="fas fa-exclamation-circle"></i> Error: ${data.error}`;
                }
            } catch (error) {
                console.error('Polling error:', error);
            }
        }, 2000);
    }

    stopPolling() {
        if (this.pollingInterval) {
            clearInterval(this.pollingInterval);
            this.pollingInterval = null;
        }
    }

    showDashboard() {
        document.getElementById('code-section').classList.add('hidden');
        document.getElementById('dashboard-section').classList.remove('hidden');
        window.dashboardManager.loadStatus();
    }

    reset() {
        this.stopPolling();
        document.getElementById('phone-input').value = '';
        document.getElementById('phone-error').textContent = '';
        document.getElementById('dashboard-section').classList.add('hidden');
        document.getElementById('code-section').classList.add('hidden');
        document.getElementById('phone-section').classList.remove('hidden');
    }

    promptForApiKey() {
        const key = prompt('Enter API key:');
        if (key) {
            localStorage.setItem('api_key', key);
            this.apiKey = key;
        }
    }
}

const pairingManager = new PairingManager();

document.getElementById('generate-btn').addEventListener('click', () => {
    pairingManager.generateCode();
});

document.getElementById('new-pairing-btn').addEventListener('click', () => {
    pairingManager.reset();
});
