# 📍 GMapScraper & Bulk Emailer 🚀

A high-performance, automated lead generation and marketing platform. Extract high-quality business leads directly from Google Maps and dispatch personalized bulk email campaigns through custom SMTP relays—all within a stunning, modern dashboard.

![GitHub License](https://img.shields.io/badge/license-MIT-blue.svg)
![Maintained](https://img.shields.io/badge/Maintained-Yes-green.svg)
![Svelte 5](https://img.shields.io/badge/Frontend-Svelte%205-orange.svg)
![Go](https://img.shields.io/badge/Backend-Go-blue.svg)

---

## ✨ Key Features

### 🔍 Powerful Extraction Engine
- **Multi-Keyword Scrapes**: Run multiple search queries simultaneously.
- **Deep Lead Enrichment**: Automatically extracts business names, physical addresses, ratings, and crucially—**contact phone numbers and email addresses**.
- **Playwright Powered**: Native browser automation for reliable and undetectable extraction.

### 📧 Integrated Bulk Emailer
- **Custom SMTP Routing**: Connect multiple mail servers (Gmail, Mailgun, SendGrid, etc.) and switch between them instantly.
- **Smart Validation**: Real-time Regex validation with active **media filtering** (automatically detects and removes logo/retina assets masquerading as emails).
- **Rich Text Editor**: Compose elegant HTML emails with a built-in Quill.js editor.
- **In-Field Highlighting**: Visual error feedback directly in the targets input field.

### 📊 Professional Management
- **Persistent History**: View and manage all previous scraping jobs in a searchable history log.
- **Quick Exports**: Single-click CSV export and clipboard copying for filtered leads.
- **Glassmorphism UI**: A premium, responsive dark-themed dashboard built with Svelte 5.

---

## 🚀 Quick Start (Docker)

The fastest and most reliable way to deploy the platform is via Docker Compose.

1.  **Clone the Repository**
    ```bash
    git clone https://github.com/adnansamirswe/google-maps-scraper-and-bulk-email-sender.git
    cd google-maps-scraper-and-bulk-email-sender
    ```

2.  **Start the Services**
    ```bash
    docker compose up -d --build
    ```

3.  **Access the Dashboard**
    - **Frontend**: [http://localhost:4205](http://localhost:4205)
    - **Backend API**: [http://localhost:4206](http://localhost:4206)

---

## 🛠️ Manual Installation

### Backend Setup (Go)
1. Navigate to the backend directory: `cd backend`
2. Install dependencies: `go mod download`
3. Configure environment in `.env`:
   ```env
   PORT=4206
   APP_PASSWORD=your_secure_password
   JWT_SECRET=your_jwt_secret
   ```
4. Run the server: `go run main.go`

### Frontend Setup (Svelte 5)
1. Navigate to the frontend directory: `cd frontend`
2. Install dependencies: `npm install`
3. Configure API target in `.env`:
   ```env
   VITE_API_BASE=http://localhost:4206
   ```
4. Start the dev server: `npm run dev`

---

## ⚙️ Configuration

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | Backend API port | `3001` (Docker: `4206`) |
| `APP_PASSWORD` | Master password for dashboard access | `admin123` |
| `JWT_SECRET` | Secret key for session encryption | `default-secret` |
| `DATA_DIR` | Local directory for SQLite database | `./data` |

---

## 📄 License
This project is licensed under the **MIT License**. See the [LICENSE](LICENSE) file for details.

## 🤝 Contributing
Contributions are welcome! Please feel free to submit a Pull Request or open an issue for feature requests.

---

*Built with ❤️ for High-Performance Marketing Automation.*
