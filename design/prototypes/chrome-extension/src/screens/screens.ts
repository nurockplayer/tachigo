export function showScreen(screenMap, activeName) {
  Object.entries(screenMap).forEach(([name, element]) => {
    element.hidden = name !== activeName;
  });
}

export function wireLoginModes({ forgotButton, form, passwordField, signupButton, signupCopy }, onLogin) {
  let mode = "login";

  function renderMode() {
    const isSignup = mode === "signup";
    const isForgot = mode === "forgot";
    passwordField.hidden = isForgot;
    passwordField.querySelector("input").required = !isForgot;
    signupCopy.textContent = isSignup ? "Already have an account?" : "Don't have an account?";
    signupButton.textContent = isSignup ? "LOGIN" : "SIGN UP";
  }

  forgotButton.addEventListener("click", () => {
    mode = "forgot";
    renderMode();
  });

  signupButton.addEventListener("click", () => {
    mode = mode === "signup" ? "login" : "signup";
    renderMode();
  });

  form.addEventListener("submit", (event) => {
    event.preventDefault();
    if (mode === "forgot") {
      mode = "login";
      renderMode();
      return;
    }
    onLogin();
  });

  renderMode();
}
