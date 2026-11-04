async function login() {
  const resp = await fetch(`${baseURL}/internal/login/challenge`);
  const respBody = await resp.json();

  const publicKey = respBody.data.publicKey;
  publicKey.challenge = Uint8Array.fromBase64(publicKey.challenge, {
    alphabet: "base64url",
  });
  if (publicKey.allowCredentials) {
    publicKey.allowCredentials.forEach((credential) => {
      credential.id = Uint8Array.fromBase64(credential.id, {
        alphabet: "base64url",
      });
    });
  }

  console.log(publicKey);

  const result = await navigator.credentials.get({ publicKey });
  console.log(result);

  const assertionResp = await fetch(`${baseURL}/internal/login/finish`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      // fidoToJSON is needed, because of this bug: https://github.com/microsoft/playwright/issues/43093
      // should be fixed in the next release of playwright
      assertion: JSON.stringify(fidoToJSON(result, "get")),
    }),
  });

  let auth = await assertionResp.json();
  let authToken = auth.auth?.client_token;
  if (authToken !== undefined) {
    console.log(`success`, authToken);
  } else {
    console.error("failed", authToken, auth);
    if (!auth.errors && auth.errors.length > 0) {
      throw new Error("unknown error");
    }
    if (auth.errors.length == 1) {
      throw new Error(auth.errors[0]);
    }
    throw new Error("Multiple Errors:\n  - " + auth.errors.join("\n  - "));
  }

  let fieldClientToken = document.getElementById("client-token");
  fieldClientToken.value = authToken;
  document.getElementById("client-token-card").style.display = "";

  lastAuth = auth;
}

function init() {
  const btnEnroll = document.getElementById("btn-login");
  btnEnroll.addEventListener("click", errorWrap(login));

  initCommon();
}

window.onload = init;
