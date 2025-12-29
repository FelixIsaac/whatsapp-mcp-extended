class DashboardManager {
    constructor() {
        const currentProtocol = window.location.protocol;
        const currentHost = window.location.hostname;
        this.apiBaseUrl = `${currentProtocol}//${currentHost}:8180/api`;
        this.apiKey = localStorage.getItem('api_key');
    }

    async loadStatus() {
        try {
            const response = await fetch(`${this.apiBaseUrl}/connection`, {
                headers: { 'X-API-Key': this.apiKey }
            });
            const data = await response.json();

            if (data.success && data.linked) {
                document.getElementById('jid-text').textContent = data.jid;
            } else {
                document.getElementById('jid-text').textContent = 'Not linked';
            }
        } catch (error) {
            document.getElementById('jid-text').textContent = 'Error loading status';
        }
    }
}

// Initialize
const dashboardManager = new DashboardManager();

document.getElementById('refresh-btn').addEventListener('click', () => {
    dashboardManager.loadStatus();
});

// Auto-load dashboard on page load if already connected
window.addEventListener('DOMContentLoaded', async () => {
    const apiKey = localStorage.getItem('api_key');
    if (apiKey) {
        try {
            const response = await fetch(`${window.location.protocol}//${window.location.hostname}:8180/api/connection`, {
                headers: { 'X-API-Key': apiKey }
            });
            const data = await response.json();

            if (data.success && data.linked) {
                document.getElementById('phone-section').classList.add('hidden');
                document.getElementById('dashboard-section').classList.remove('hidden');
                dashboardManager.loadStatus();
            }
        } catch (error) {
            console.error('Failed to check connection status:', error);
        }
    }
});
