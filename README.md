# Badminton Backend

This is the backend for the Badminton application, built using **Golang** and **Echo framework**. It provides APIs for user authentication, session management, and more. The backend is designed to be deployed to **AWS Lambda** and uses **Amazon RDS (PostgreSQL)** as the database.

---

## Table of Contents

1. [Features](#features)
2. [Technologies](#technologies)
3. [Setup](#setup)
   - [Prerequisites](#prerequisites)
   - [Local Development](#local-development)
   - [Login with Google Setup](#login-with-google-setup)
   - [Deployment to AWS Lambda](#deployment-to-aws-lambda)
4. [API Documentation](#api-documentation)
5. [Environment Variables](#environment-variables)
6. [Contributing](#contributing)
7. [License](#license)

---

## Features

- **User Authentication**:
  - Google OAuth2 login.
  - JWT-based authentication for protected routes.
- **Session Management**:
  - Create, update, and delete badminton sessions.
  - Allow users to attend sessions.
- **Admin Role**:
  - Admins can create and manage sessions.
- **Swagger Documentation**:
  - Automatically generated API documentation using **go-swagger**.

---

## Technologies

- **Golang**: Backend programming language.
- **Echo**: High-performance web framework for Golang.
- **GORM**: ORM for database interactions.
- **PostgreSQL**: Database for storing user and session data.
- **AWS Lambda**: Serverless deployment platform.
- **Amazon RDS**: Managed PostgreSQL database service.
- **go-swagger**: Swagger documentation generator for Golang.

---

## Setup

### Prerequisites

- **Go** (v1.20 or higher)
- **PostgreSQL** (local or Amazon RDS)
- **AWS CLI** (for deployment)
- **Serverless Framework** (for deployment to AWS Lambda)
- **Docker** (optional, for local PostgreSQL)

---

### Local Development

1. **Clone the repository**:
   ```bash
   git clone https://github.com/alanrb/badminton.git
   cd badminton/backend
   ```

2. **Set up environment variables**:
   Create a `.env` file in the root directory with the following variables:
   ```env
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=yourpassword
   DB_NAME=badminton_db
   JWT_SECRET=your_jwt_secret_key
   GOOGLE_CLIENT_ID=your_google_client_id
   GOOGLE_CLIENT_SECRET=your_google_client_secret
   ```

3. **Start PostgreSQL**:
   Use Docker to run a local PostgreSQL instance:
   ```bash
   docker-compose up -d
   ```

4. **Run migrations**:
   The application will automatically migrate the database schema on startup.

5. **Start the server**:
   ```bash
   go run main.go
   ```

6. **Access the API**:
   - Swagger UI: `http://localhost:8080/swagger`
   - API Base URL: `http://localhost:8080`

---

### Login with Google Setup

To enable **Login with Google**, follow these steps:

1. **Create a Google OAuth2 Client**:
   - Go to the [Google Cloud Console](https://console.cloud.google.com/).
   - Create a new project (if you don't already have one).
   - Navigate to **APIs & Services > Credentials**.
   - Click **Create Credentials > OAuth Client ID**.
   - Choose **Web Application** as the application type.
   - Add the following authorized redirect URIs:
     - `http://localhost:8080/auth/google/callback`
   - Save the **Client ID** and **Client Secret**.

2. **Update Environment Variables**:
   Update the `.env` file with your Google OAuth2 credentials:
   ```env
   GOOGLE_CLIENT_ID=your_google_client_id
   GOOGLE_CLIENT_SECRET=your_google_client_secret
   ```

3. **Test Google Login**:
   - Start the server:
     ```bash
     go run main.go
     ```
   - Navigate to `http://localhost:8080/auth/google/login` in your browser.
   - You will be redirected to Google's OAuth2 login page.
   - After logging in, you will be redirected back to the callback URL with a JWT token.

---

### Deployment to AWS Lambda

1. **Install Serverless Framework**:
   ```bash
   npm install -g serverless
   ```

2. **Configure AWS credentials**:
   ```bash
   aws configure
   ```

3. **Deploy the backend**:
   ```bash
   serverless deploy
   ```

4. **Access the deployed API**:
   The Serverless Framework will output the API Gateway endpoint. Use this endpoint to access the API and Swagger UI.

---

## API Documentation

The API documentation is automatically generated using **go-swagger** and is available at:
- **Local**: `http://localhost:8080/swagger`
- **Deployed**: `https://<api-gateway-id>.execute-api.<region>.amazonaws.com/dev/swagger`

---

## Environment Variables

| Variable            | Description                          | Example Value                |
|---------------------|--------------------------------------|------------------------------|
| `DB_HOST`           | PostgreSQL host                      | `localhost` or RDS endpoint  |
| `DB_PORT`           | PostgreSQL port                      | `5432`                       |
| `DB_USER`           | PostgreSQL username                  | `postgres`                   |
| `DB_PASSWORD`       | PostgreSQL password                  | `yourpassword`               |
| `DB_NAME`           | PostgreSQL database name             | `badminton_db`               |
| `JWT_SECRET`        | Secret key for JWT tokens            | `your_jwt_secret_key`        |
| `GOOGLE_CLIENT_ID`  | Google OAuth2 client ID              | `your_google_client_id`      |
| `GOOGLE_CLIENT_SECRET` | Google OAuth2 client secret       | `your_google_client_secret`  |

---

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository.
2. Create a new branch: `git checkout -b feature/your-feature-name`.
3. Commit your changes: `git commit -m "Add your feature"`.
4. Push to the branch: `git push origin feature/your-feature-name`.
5. Submit a pull request.

---

## License

This project is licensed under the **MIT License**. See the [LICENSE](LICENSE) file for details.
