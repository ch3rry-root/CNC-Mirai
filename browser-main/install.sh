#!/bin/bash

echo "Updating system..."
apt update -y

echo "Installing base packages..."
apt install -y wget gnupg curl git build-essential

echo "Installing Chrome dependencies..."
apt install -y \
  libatk1.0-0 \
  libatk-bridge2.0-0 \
  libgtk-3-0 \
  libnss3 \
  libxss1 \
  libgbm1 \
  libx11-xcb1 \
  libxcomposite1 \
  libxdamage1 \
  libxrandr2 \
  libxfixes3 \
  libxext6 \
  libxi6 \
  libxcursor1 \
  libpangocairo-1.0-0 \
  libpango-1.0-0 \
  libcairo2 \
  libcups2 \
  libdbus-1-3 \
  libexpat1 \
  libfontconfig1 || true

# ── Node.js ───────────────────────────────────────────────────────────────────
echo ""
echo "Checking for Node.js..."
if command -v node &>/dev/null; then
  echo "Node.js already installed: $(node -v)"
else
  echo "Node.js not found. Installing..."
  curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -
  apt install -y nodejs
  echo "Node.js installed: $(node -v)"
fi

echo "Checking for npm..."
if command -v npm &>/dev/null; then
  echo "npm already installed: $(npm -v)"
else
  apt install -y npm
fi

# ── Google Chrome ─────────────────────────────────────────────────────────────
echo ""
echo "Checking for Google Chrome..."
if command -v google-chrome &>/dev/null || command -v google-chrome-stable &>/dev/null; then
  echo "Google Chrome already installed."
else
  echo "Installing Google Chrome..."
  wget -q https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
  apt install -y ./google-chrome-stable_current_amd64.deb
  rm -f google-chrome-stable_current_amd64.deb
fi

# ── npm packages ──────────────────────────────────────────────────────────────
echo ""
echo "Installing required npm packages..."
PACKAGES="puppeteer puppeteer-extra puppeteer-extra-plugin-stealth async request hpack"

for pkg in $PACKAGES; do
  # Use node itself to check if module resolves — most reliable method
  if node -e "require('$pkg')" &>/dev/null 2>&1; then
    echo "  ✔ $pkg already installed"
  else
    echo "  Installing $pkg..."
    npm install "$pkg" && echo "  ✔ $pkg installed" || echo "  ⚠ Failed to install $pkg"
  fi
done

# ── Run browser.js + auto-install missing modules ─────────────────────────────
echo ""
echo "========================================"
echo " Running node browser.js ..."
echo "========================================"

if [ ! -f "browser.js" ]; then
  echo "✘ browser.js not found in current directory: $(pwd)"
  echo "  Please make sure browser.js is in the same folder as this script."
  exit 1
fi

MAX_RETRIES=10
attempt=0

while [ $attempt -lt $MAX_RETRIES ]; do
  attempt=$((attempt + 1))
  echo ""
  echo "--- Attempt $attempt ---"

  OUTPUT=$(node browser.js 2>&1)
  EXIT_CODE=$?

  echo "$OUTPUT"

  if [ $EXIT_CODE -eq 0 ]; then
    echo ""
    echo "✅ browser.js exited successfully."
    exit 0
  fi

  # Detect missing module
  MISSING=$(echo "$OUTPUT" | grep -oP "(?<=Cannot find module ')([^']+)" | head -1)

  if [ -n "$MISSING" ]; then
    echo ""
    echo "⚠ Missing module: '$MISSING' — installing..."
    if npm install "$MISSING"; then
      echo "✔ Installed '$MISSING'. Retrying..."
    else
      echo "✘ Could not install '$MISSING'. Aborting."
      exit 1
    fi
  else
    echo ""
    echo "✘ browser.js failed (exit code $EXIT_CODE) — no missing module detected."
    echo "  Fix the error above and re-run."
    exit 1
  fi
done

echo "✘ Reached max retries ($MAX_RETRIES). Aborting."
exit 1