export const ASSETS = {
  opening01: "public/assets/first page/opening01.mp4",
  opening02: "public/assets/first page/opening02.mp4",
  logo: "public/assets/first page/logo.svg",

  loginBackground: "public/assets/login/login_background.svg",
  loginCharacter: "public/assets/login/login_character.svg",
  loginSubtitle: "public/assets/login/login_subtitle.svg",
  loginEmailInput: "public/assets/login/login_Input-email.svg",
  loginPasswordInput: "public/assets/login/login_Input-password.svg",
  loginButton: "public/assets/login/Login_Button.svg",
  loginTwitchButton: "public/assets/login/Login_button-Twitch.svg",
  loginOr: "public/assets/login/login_or.svg",

  characterTitle: "public/assets/CharacterSelect/title.svg",
  characterCrab: "public/assets/CharacterSelect/crab_character.mp4",
  characterDiveButton: "public/assets/CharacterSelect/dive-in_button.svg",
  characterLeftArrow: "public/assets/CharacterSelect/left_arrow.svg",
  characterRightArrow: "public/assets/CharacterSelect/right_arrow.svg",
  characterPageDots: "public/assets/CharacterSelect/page_icon.svg",

  crabSayHi: "public/assets/crab/say-hi/sayHi-loop.mp4",
  crabIdle: "public/assets/crab/idle/idle-loop.mp4",
  crabMining: "public/assets/crab/mining/mining-loop.mp4",

  hud: {
    arrows: "public/assets/icon_button/arrows.svg",
    cpc: "public/assets/icon_button/CPC_icon.svg",
    level: "public/assets/icon_button/level_icon.svg",
    miningSpeed: "public/assets/icon_button/mining_speed_icon.svg",
    perClick: "public/assets/icon_button/pre_click_icon.svg",
    settings: "public/assets/icon_button/setting_icon.svg"
  }
};

export function assetValue(path) {
  return path.split(".").reduce((current, key) => current?.[key], ASSETS);
}
