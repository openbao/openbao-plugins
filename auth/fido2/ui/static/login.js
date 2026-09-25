async function login() {
  const entityId = document.getElementById("entity-id").value;
  const resp = await fetch(`${baseURL}/internal/login/challenge?entity_id=${entityId}`);
  const respBody = await resp.json();

  const publicKey = respBody.data.publicKey;
  publicKey.challenge = Uint8Array.fromBase64(publicKey.challenge, {
    alphabet: "base64url",
  });
  publicKey.allowCredentials.forEach((credential) => {
    credential.id = Uint8Array.fromBase64(credential.id, {
      alphabet: "base64url",
    });
  });

  console.log(publicKey);

  const result = await navigator.credentials.get({ publicKey });
  console.log(result);

  const assertionResp = await fetch(`${baseURL}/internal/login/finish`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      assertion: JSON.stringify(result),
      entity_id: entityId,
    }),
  });

  let auth = await assertionResp.json();
  let authToken = auth.auth?.client_token;
  if (authToken !== undefined) {
    console.log(`success`, authToken);
  } else {
    console.error("failed", authToken, auth);
  }

  let fieldClientToken = document.getElementById("client-token");
  fieldClientToken.value = authToken;

  lastAuth = auth;
}

function init() {
  const btnEnroll = document.getElementById("btn-login");
  btnEnroll.addEventListener("click", login);

  initCommon();
}

window.onload = init;
