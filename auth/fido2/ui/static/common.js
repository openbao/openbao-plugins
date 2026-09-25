const TOKEN_SEPARATOR = '☃';
const TOKEN_PREFIX = 'vault-';
let lastAuth = null;

function findAPI() {
  let parts = window.location.href.split("/");
  
  while (parts[parts.length-1] != "internal" && parts.length > 3) {
    parts.pop();
  }

  parts.pop(); // pop the "internal"

  return parts.join("/");
}

const baseURL = findAPI();

function toggleClientToken() {
  let fieldClientToken = document.getElementById("client-token");
  const btnToggleClientToken = document.getElementById("btn-toggle-client-token");

  if (fieldClientToken.type === "password") {
    fieldClientToken.type = "text";
    btnToggleClientToken.innerText = "Hide Token";
  } else {
    fieldClientToken.type = "password";
    btnToggleClientToken.innerText = "Show Token";
  }
}

function copyClientToken() {
  let fieldClientToken = document.getElementById("client-token");
  navigator.clipboard.writeText(fieldClientToken.value);
}

function openUI() {
  if (lastAuth === null) {
    throw new Error("not logged in");
  }
  localStorage.setItem(`${TOKEN_PREFIX}passkey${TOKEN_SEPARATOR}1`, JSON.stringify({
    token: lastAuth.auth.client_token,
    entity_id: lastAuth.auth.entity_id,
    policies: lastAuth.auth.policies,
    renewable: lastAuth.auth.renewable,
  }));
  
  window.location.pathname = "/ui";
}

function initCommon() {
  const btnToggleClientToken = document.getElementById("btn-toggle-client-token");
  btnToggleClientToken.addEventListener("click", toggleClientToken);

  const btnCopyClientToken = document.getElementById("btn-copy-client-token");
  btnCopyClientToken.addEventListener("click", copyClientToken);

  const btnOpenUI = document.getElementById("btn-open-ui");
  btnOpenUI.addEventListener("click", openUI);
}
