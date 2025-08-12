# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WilliamBoard is an event flyer analysis tool that uses OpenAI's GPT-4 Vision to extract event details from images of flyers and posters.

## Development Commands

```bash
# Install dependencies
npm install

# Run flyer analysis on test-data images
npm run analyze
# or
npm test

# Analyze flyers directly
node analyze-flyers.js
```

## Setup Requirements

1. Set OpenAI API key:
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```

2. Place image files in `test-data/` directory

## Project Structure

- `analyze-flyers.js` - Main analysis script using OpenAI GPT-4 Vision
- `test-data/` - Directory containing sample flyer images
- `analysis-results.json` - Output file with extracted event data
- `.env.example` - Template for environment variables

## Key Features

- Processes multiple image formats (JPG, PNG, GIF, etc.)
- Extracts structured event data (date, time, location, performers, etc.)
- Outputs results in JSON format
- Saves consolidated results to file

## Architecture Notes

The script uses OpenAI's GPT-4 Vision model (gpt-4o) to analyze images and extract event information in a structured JSON format. Images are base64 encoded before sending to the API.