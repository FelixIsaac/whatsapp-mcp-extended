# WhatsApp Webhook Manager UI

A simple, modern web interface for managing WhatsApp webhook configurations. This UI provides an intuitive way to create, view, update, and delete webhooks for your WhatsApp bridge service.

## Features

- ✅ **Create Webhooks** - Add new webhook configurations with custom triggers
- 👀 **View Webhooks** - See all your existing webhooks in a clean card layout
- ✏️ **Update Webhooks** - Edit webhook settings and triggers
- 🗑️ **Delete Webhooks** - Remove unwanted webhooks
- 🔄 **Toggle Webhooks** - Enable/disable webhooks without deleting them
- 🧪 **Test Webhooks** - Send test requests to validate webhook endpoints
- 📊 **View Logs** - Check webhook delivery logs and status
- 📱 **Responsive Design** - Works on desktop and mobile devices

## Getting Started

### Prerequisites

- WhatsApp Bridge service running on port 8080 (default)
- Modern web browser with JavaScript enabled

### Installation

1. Ensure your WhatsApp Bridge service is running and accessible at `http://localhost:8080`

2. Open `index.html` in your web browser:
   ```bash
   # Option 1: Direct file opening
   open index.html
   
   # Option 2: Using a simple HTTP server (recommended)
   python3 -m http.server 3000
   # Then open http://localhost:3000
   
   # Option 3: Using Node.js http-server
   npx http-server -p 3000
   # Then open http://localhost:3000
   ```

### Configuration

The UI is configured to connect to the WhatsApp Bridge API at `http://localhost:8080/api`. If your service is running on a different host or port, update the `apiBaseUrl` in `script.js`:

```javascript
constructor() {
    this.apiBaseUrl = 'http://your-host:your-port/api';
    // ...
}
```

## Usage

### Creating a Webhook

1. Click the **"Add Webhook"** button
2. Fill in the webhook details:
   - **Name**: A descriptive name for your webhook
   - **Webhook URL**: The endpoint that will receive webhook notifications
   - **Secret Token**: (Optional) Security token for webhook validation
   - **Enabled**: Whether the webhook should be active
3. Configure triggers:
   - **All Messages**: Trigger on every message
   - **Specific Chat**: Trigger for messages from a specific chat JID
   - **Specific Sender**: Trigger for messages from a specific sender
   - **Keyword**: Trigger when messages contain specific keywords
   - **Media Type**: Trigger for specific media types (image, video, etc.)
4. Click **"Create Webhook"**

### Managing Existing Webhooks

Each webhook card provides action buttons:

- **🔄 Test**: Send a test request to the webhook URL
- **📋 Logs**: View recent webhook delivery logs
- **✏️ Edit**: Modify webhook configuration
- **🔘 Toggle**: Enable or disable the webhook
- **🗑️ Delete**: Remove the webhook (with confirmation)

### Webhook Triggers

You can configure multiple triggers for each webhook:

- **Trigger Type**: What kind of event should trigger the webhook
- **Trigger Value**: The specific value to match (not needed for "All Messages")
- **Match Type**: How to match the trigger value:
  - **Exact**: Exact string match
  - **Contains**: Check if the value is contained in the message
  - **Regex**: Use regular expression matching

### Tips

- Use meaningful names for your webhooks to easily identify them
- Test your webhooks after creation to ensure they're working correctly
- Check the logs if webhooks aren't being triggered as expected
- Use the "All Messages" trigger type for development and testing
- Disable webhooks instead of deleting them if you might need them later

## API Endpoints

The UI interacts with the following WhatsApp Bridge API endpoints:

- `GET /api/webhooks` - List all webhooks
- `POST /api/webhooks` - Create new webhook
- `GET /api/webhooks/{id}` - Get specific webhook
- `PUT /api/webhooks/{id}` - Update webhook
- `DELETE /api/webhooks/{id}` - Delete webhook
- `POST /api/webhooks/{id}/test` - Test webhook
- `GET /api/webhooks/{id}/logs` - Get webhook logs
- `POST /api/webhooks/{id}/enable` - Enable/disable webhook

## Troubleshooting

### Common Issues

1. **"Failed to load webhooks"**
   - Check that the WhatsApp Bridge service is running
   - Verify the API URL in the browser console
   - Ensure CORS is properly configured on the backend

2. **Webhook test failures**
   - Verify the webhook URL is accessible
   - Check that the target endpoint accepts POST requests
   - Review webhook logs for detailed error information

3. **UI not loading properly**
   - Ensure all files (HTML, CSS, JS) are in the same directory
   - Check browser console for JavaScript errors
   - Try serving the files through an HTTP server instead of opening directly

### Browser Console

Open your browser's developer tools (F12) to see detailed error messages and logs that can help diagnose issues.

## Development

### File Structure

```
whatsapp-webhook-ui/
├── index.html          # Main HTML structure
├── styles.css          # CSS styling and responsive design
├── script.js           # JavaScript application logic
└── README.md          # This documentation
```

### Customization

- **Styling**: Modify `styles.css` to change colors, fonts, and layout
- **API URL**: Update `apiBaseUrl` in `script.js` for different backend locations
- **Features**: Extend `script.js` to add new functionality

## Browser Support

- Chrome 60+
- Firefox 55+
- Safari 12+
- Edge 79+

## License

This project is part of the WhatsApp MCP Bridge system. See the main project license for details.
