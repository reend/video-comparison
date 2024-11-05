# Video Comparison Project

This is a Go-based project designed to compare videos in Google Cloud Run. The application provides a set of HTTP endpoints to manage video downloads and comparisons, ensuring that all videos are stored in memory for quick access and processing.

## Overview

The project includes the following key functionalities:

- **Video Downloading**: Downloads videos from provided URLs and stores them in memory.
- **Video Comparison**: Compares a main video with a set of downloaded videos to identify identical ones.
- **Trigger Endpoint**: Keeps the application alive on Google Cloud cron job to ensure all videos are stored in memory, because Google Cloud will kill released resources if it's not used for a while.

## Endpoints

### `/download`

- **Method**: POST
- **Description**: Downloads videos from the provided URLs and stores them in memory. This endpoint should be called with a list of video data to initiate the download process.

### `/compare`

- **Method**: POST
- **Description**: Compares a main video with the videos stored in memory. If the videos are not fully downloaded (`isDownloaded` is false), the client should first call the `/download` endpoint with all video data. Once the download is complete and a `200 OK` response is received, the client can proceed to call `/compare`.

### `/trigger`

- **Method**: GET
- **Description**: Keeps the application alive on Google Cloud by ensuring all videos are stored in memory. This endpoint is crucial for maintaining the state of the application.

## Usage

1. **Trigger the Application**: Use the `/trigger` on cron job to ensure the application is running and all videos are stored in memory.

2. **Download Videos**: If you need to compare videos, first call the `/download` endpoint with the necessary video data. Wait for a `200 OK` response to confirm that all videos have been successfully downloaded.

3. **Compare Videos**: Once the videos are downloaded, use the `/compare` endpoint to compare the main video with the stored videos. Ensure that `isDownloaded` is true before making this request.

## Deployment

This project is designed to run on Google Cloud, utilizing in-memory storage for efficient video processing. Ensure that the application is properly configured to handle requests and maintain its state across sessions.

## Requirements

- Go 1.20+
- Docker (for containerization)

## Building and Running

To build and run the project, use the provided `Dockerfile`:

```bash
docker build -t video-comparison .
docker run -p 8080:8080 video-comparison
```

This will start the server on port 8080, ready to handle video download and comparison requests.
