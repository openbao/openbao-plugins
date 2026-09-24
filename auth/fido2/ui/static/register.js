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
}

function toggleClientToken() {
  let fieldClientToken = document.getElementById("client-token");
  if (fieldClientToken.type === "password") {
    fieldClientToken.type = "text";
    fieldClientToken.innerText = "Hide Client Token";
  } else {
    fieldClientToken.type = "password";
    fieldClientToken.innerText = "Show Client Token";
  }
}

function copyClientToken() {
  let fieldClientToken = document.getElementById("client-token");
  navigator.clipboard.writeText(fieldClientToken.value);
}

function init() {
  const btnEnroll = document.getElementById("btn-enroll");
  btnEnroll.addEventListener("click", enroll);

  const btnShowClientToken = document.getElementById("btn-toggle-client-token");
  btnShowClientToken.addEventListener("click", toggleClientToken);

  const btnCopyClientToken = document.getElementById("btn-copy-client-token");
  btnCopyClientToken.addEventListener("click", copyClientToken);
}

window.onload = init;
