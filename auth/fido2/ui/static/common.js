const TOKEN_SEPARATOR = '☃';
const TOKEN_PREFIX = 'vault-';
let lastAuth = null;

function findAPI() {
  let parts = window.location.pathname.split("/");
  
  while (parts[parts.length-1] != "internal" && parts.length > 3) {
    parts.pop();
  }

  parts.pop(); // pop the "internal"

  return parts.join("/");
}

const baseURL = findAPI();

function localStorageKey(name) {
  return `fido2-auth-plugin@${baseURL} ${name}`;
}

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

async function checkedFetch() {
  const resp = await fetch(...arguments);
  if (!resp.ok) {
    const j = await resp.json();
    if (!j.errors || !j.errors.length || j.errors.length === 0) {
      throw new Error(`Unexpected HTTP status ${resp.status}`);
    }
    if (j.errors.length > 1) {
      throw new Error(`Got multiple errors: ${j.errors.join(", ")}`);
    }
    throw new Error(j.errors[0]);
  }
  return resp;
}

function errorWrap(f) {
  return async function () {
    const divErrorCard = document.getElementById("error-card");
    const spanErrorText = document.getElementById("error-text");
    divErrorCard.style.display = "none";
    document.getElementById("client-token-card").style.display = "none";

    try {
      return await f(...arguments);
    } catch (e) {
      console.error("caught error", e);
      spanErrorText.innerText = e.message;
      divErrorCard.style.display = "";
    }
  }
}

function fidoToJSON(x, getOrCreate) {
  b64 = (a) => {
    return new Uint8Array(a).toBase64({
      alphabet: "base64url",
      omitPadding: true,
    });
  };

  let response ={
    "authenticatorData": b64(x.response.authenticatorData),
    "clientDataJSON": b64(x.response.clientDataJSON),
  }

  switch (getOrCreate) {
    case "get":
        response["signature"]= b64(x.response.signature);
        response["userHandle"]= b64(x.response.userHandle);
          break;

    case "create":
      response["attestationObject"] = b64(x.response.attestationObject);
      response["clientDataJSON"] = b64(x.response.clientDataJSON);
      response["publicKey"] = b64(x.response.getPublicKey());
      response["publicKeyAlgorithm"] = x.response.getPublicKeyAlgorithm();
      response["transports"] = x.response.getTransports();
      break;

    default:
      throw new Error(`expected "get" or "create" got "${getOrCreate}"`);
  }

  return {
      "authenticatorAttachment": x.authenticatorAttachment,
      "clientExtensionResults": {},
      "id": x.id,
      "rawId": x.id,
      "response": response,
      "type": x.type,
  }
}
