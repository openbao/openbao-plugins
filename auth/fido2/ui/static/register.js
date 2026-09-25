async function enroll() {
  const token = document.getElementById("token").value;
  const resp = await fetch(
    `${baseURL}/internal/enroll/challenge?token=${token}`,
  );
  const respBody = await resp.json();

  const publicKey = respBody.data.publicKey;
  publicKey.challenge = Uint8Array.fromBase64(publicKey.challenge, {
    alphabet: "base64url",
  });
  publicKey.user.id = Uint8Array.fromBase64(publicKey.user.id, {
    alphabet: "base64url",
  });
  publicKey.excludeCredentials.forEach((credential) => {
    credential.id = Uint8Array.fromBase64(credential.id, {
      alphabet: "base64url",
    });
  });


  const enrollResult = await navigator.credentials.create({ publicKey });

  const assertionResp = await fetch(`${baseURL}/internal/enroll/assertion`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      assertion: JSON.stringify(enrollResult),
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
  document.getElementById("client-token-card").style.display = ""

  localStorage.setItem(localStorageKey("alias"), auth.data.alias);

  lastAuth = auth;
}

function init() {
  const btnEnroll = document.getElementById("btn-enroll");
  btnEnroll.addEventListener("click", enroll);

  initCommon();
}

window.onload = init;
