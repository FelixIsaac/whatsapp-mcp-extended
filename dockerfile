# Dockerfile for WhatsApp MCP Server
FROM python:3.13 AS runtime

WORKDIR /app

# Copy the Python Gradio project files
COPY ./whatsapp-mcp-server /app/whatsapp-mcp-server

# Install Python and other dependencies
RUN apt-get update 
RUN apt-get install -y ffmpeg \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# Set up Python environment
RUN python3 -m pip install --upgrade pip
RUN python3 -m pip install uv

# Install dependencies from requirements.txt
WORKDIR /app/whatsapp-mcp-server
COPY ./whatsapp-mcp-server/requirements.txt /app/whatsapp-mcp-server/requirements.txt
RUN pip install -r requirements.txt

# Set up Python MCP server
WORKDIR /app/whatsapp-mcp-server

# Create directories for persistent storage
RUN mkdir -p /app/store

# Create entrypoint script for MCP server
RUN echo '#!/bin/bash\ncd /app/whatsapp-mcp-server\npython gradio-main.py' > /app/entrypoint-mcp.sh
RUN chmod +x /app/entrypoint-mcp.sh

# Expose ports for MCP server and Gradio UI
EXPOSE 8081
EXPOSE 8082

ENTRYPOINT ["/app/entrypoint-mcp.sh"]