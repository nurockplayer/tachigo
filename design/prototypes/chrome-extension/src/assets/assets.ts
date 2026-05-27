export const ASSETS = {
  opening01: "public/assets/first-page/opening-01.mp4",
  opening02: "public/assets/first-page/opening-02.mp4",
  logo: "public/assets/first-page/logo.svg",

  loginBackground: "public/assets/login/login-background.svg",
  loginCharacter: "public/assets/login/login-character.svg",
  loginSubtitle: "public/assets/login/login-subtitle.svg",
  loginEmailInput: "public/assets/login/login-input-email.svg",
  loginPasswordInput: "public/assets/login/login-input-password.svg",
  loginButton: "public/assets/login/login-button.svg",
  loginTwitchButton: "public/assets/login/login-button-twitch.svg",
  loginOr: "public/assets/login/login-or.svg",

  characterTitle: "public/assets/character-select/title.svg",
  characterCrab: "public/assets/character-select/crab-character-loop.mp4",
  characterDiveButton: "public/assets/character-select/dive-in-button.svg",
  characterLeftArrow: "public/assets/character-select/left-arrow.svg",
  characterRightArrow: "public/assets/character-select/right-arrow.svg",
  characterPageDots: "public/assets/character-select/page-icon.svg",

  crabSayHi: "public/assets/crab/say-hi/say-hi-loop.mp4",
  crabIdle: "public/assets/crab/idle/idle-loop.mp4",
  crabMining: "public/assets/crab/mining/mining-loop.mp4",

  hud: {
    arrows: "public/assets/icon-button/arrows.svg",
    cpc: "public/assets/icon-button/cpc-icon.svg",
    level: "public/assets/icon-button/level-icon.svg",
    miningSpeed: "public/assets/icon-button/mining-speed-icon.svg",
    perClick: "public/assets/icon-button/pre-click-icon.svg",
    settings: "public/assets/icon-button/setting-icon.svg"
  }
};

export function assetValue(path) {
  return path.split(".").reduce((current, key) => current?.[key], ASSETS);
}
