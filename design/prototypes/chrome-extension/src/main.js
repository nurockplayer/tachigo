import { ASSETS, assetValue } from "./assets/assets.js";
import { createOpeningLoop } from "./hooks/openingLoop.js";
import { showScreen, wireLoginModes } from "./screens/screens.js";

const elements = {
  openingScreen: document.querySelector("#openingScreen"),
  loginScreen: document.querySelector("#loginScreen"),
  characterScreen: document.querySelector("#characterScreen"),
  miningScreen: document.querySelector("#miningScreen"),
  opening01: document.querySelector("#opening01"),
  opening02: document.querySelector("#opening02"),
  openingLogoButton: document.querySelector("#openingLogoButton"),
  loginForm: document.querySelector("#loginForm"),
  forgotButton: document.querySelector("#forgotButton"),
  passwordField: document.querySelector("#passwordField"),
  signupButton: document.querySelector("#signupButton"),
  signupCopy: document.querySelector("#signupCopy"),
  twitchLoginButton: document.querySelector("#twitchLoginButton"),
  characterVideo: document.querySelector("#characterVideo"),
  diveInButton: document.querySelector("#diveInButton"),
  miningVideo: document.querySelector("#miningVideo"),
  mineButton: document.querySelector("#mineButton"),
  tapFeedback: document.querySelector("#tapFeedback"),
  totalMined: document.querySelector("#totalMined"),
  srStatus: document.querySelector("#srStatus")
};

const screens = {
  opening: elements.openingScreen,
  login: elements.loginScreen,
  character: elements.characterScreen,
  mining: elements.miningScreen
};

let miningState = "sayHi";
let totalMined = 12450;
let idleTimer = 0;

function setStatus(message) {
  elements.srStatus.textContent = message;
}

function applyAssets() {
  document.querySelectorAll("[data-asset]").forEach((element) => {
    const value = assetValue(element.dataset.asset);
    if (!value) return;
    element.src = value;
  });

  elements.opening01.src = ASSETS.opening01;
  elements.opening02.src = ASSETS.opening02;
  elements.characterVideo.src = ASSETS.characterCrab;
  elements.miningVideo.src = ASSETS.crabSayHi;
}

function playVideo(video, src, { loop = true, restart = true } = {}) {
  video.loop = loop;
  video.muted = true;
  video.playsInline = true;
  if (!video.currentSrc.endsWith(src)) video.src = src;
  if (restart) {
    try {
      video.currentTime = 0;
    } catch {
      // Browser may reject early seeking.
    }
  }
  video.play().catch(() => setStatus("Tap the screen to start playback."));
}

function goToLogin() {
  openingLoop.stop();
  showScreen(screens, "login");
  setStatus("Login screen.");
}

function goToCharacter() {
  showScreen(screens, "character");
  playVideo(elements.characterVideo, ASSETS.characterCrab);
  setStatus("Choose your character.");
}

function goToMining() {
  elements.characterVideo.pause();
  showScreen(screens, "mining");
  miningState = "sayHi";
  playVideo(elements.miningVideo, ASSETS.crabSayHi, { loop: false });
  setStatus("Crab says hi.");
}

function playIdle() {
  miningState = "idle";
  playVideo(elements.miningVideo, ASSETS.crabIdle);
  setStatus("Crab is idle.");
}

function mine() {
  totalMined += 1;
  elements.totalMined.textContent = totalMined.toLocaleString("en-US");
  miningState = "mining";
  playVideo(elements.miningVideo, ASSETS.crabMining);
  elements.tapFeedback.classList.remove("is-visible");
  void elements.tapFeedback.offsetWidth;
  elements.tapFeedback.classList.add("is-visible");
  window.clearTimeout(idleTimer);
  idleTimer = window.setTimeout(playIdle, 1100);
  setStatus("Mining.");
}

applyAssets();

const openingLoop = createOpeningLoop({
  opening01: elements.opening01,
  opening02: elements.opening02,
  onStatus: setStatus
});

elements.openingLogoButton.addEventListener("click", goToLogin);
wireLoginModes(
  {
    forgotButton: elements.forgotButton,
    form: elements.loginForm,
    passwordField: elements.passwordField,
    signupButton: elements.signupButton,
    signupCopy: elements.signupCopy
  },
  goToCharacter
);
elements.twitchLoginButton.addEventListener("click", goToCharacter);
elements.diveInButton.addEventListener("click", goToMining);
elements.mineButton.addEventListener("click", mine);
elements.miningVideo.addEventListener("ended", () => {
  if (miningState === "sayHi") playIdle();
});

window.addEventListener("pagehide", () => {
  openingLoop.stop();
  window.clearTimeout(idleTimer);
});

openingLoop.start();
