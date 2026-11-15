async function enrollToken() {
  return await enrollCommon(document.getElementById("token").value);
}
async function enrollCommon(token) {
  const resp = await checkedFetch(
    `${baseURL}/internal/enroll/challenge?token=${encodeURIComponent(token)}`,
  );
  const respBody = await resp.json();

  const publicKey = respBody.data.publicKey;
  publicKey.challenge = Uint8Array.fromBase64(publicKey.challenge, {
    alphabet: "base64url",
  });
  publicKey.user.id = Uint8Array.fromBase64(publicKey.user.id, {
    alphabet: "base64url",
  });
  if (publicKey.excludeCredentials) {
    publicKey.excludeCredentials.forEach((credential) => {
      credential.id = Uint8Array.fromBase64(credential.id, {
        alphabet: "base64url",
      });
    });
  }

  let enrollResult;
  let tryAgain = true;

  while (tryAgain) {
    tryAgain = false;
    enrollResult = await navigator.credentials.create({ publicKey });

    if (!respBody.data.recommend_hardware_token) {
      break;
    }

    const transports = enrollResult.response.getTransports();
    console.debug("key transports:", transports);
    if (transports.find(x => ["internal", "hybrid"].includes(x))) {
      switch (await hardwareKeyDialog()) {
        case "abort":
          removeKey(publicKey, enrollResult);
          return;
        case "continue":
          tryAgain = false;
          break;
        case "enroll-other":
          removeKey(publicKey, enrollResult);
          tryAgain = true;
          break;
      }
    }
  }

  const assertionResp = await checkedFetch(`${baseURL}/internal/enroll/assertion`, {
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
  document.getElementById("client-token-card").style.display = "";

  lastAuth = auth;
}

async function enrollSelfService() {
  const selfServiceInfo = getSelfServiceInfo();
  if (!selfServiceInfo) {
    throw new Error("not logged in");
  }

  const resp = await checkedFetch(
    `${baseURL}/self-service/enroll`, {
      method: "PUT",
      headers: {
        "X-Vault-Token": selfServiceInfo.token,
      }
    }
  );
  const respBody = await resp.json();

  return enrollCommon(respBody.data.token);
}

function reconcileSelfService() {
  const selfServiceInfo = getSelfServiceInfo();
  const btnEnroll = document.getElementById("btn-self-service-enroll");
  const spanSelfServiceUser = document.getElementById("self-service-user");
  if (!selfServiceInfo) {
    btnEnroll.style.display = "none";
    spanSelfServiceUser.innerText = "Not logged in: Login using an existing user and come back to this page";
    return;
  }

  btnEnroll.style.display = "";
  spanSelfServiceUser.innerText = `Currently logged in as "${selfServiceInfo.displayName}" (entity id "${selfServiceInfo.entity_id}")`;
}

function init() {
  const btnTokenEnroll = document.getElementById("btn-token-enroll");
  btnTokenEnroll.addEventListener("click", errorWrap(enrollToken));

  const btnSelfServiceEnroll = document.getElementById("btn-self-service-enroll");
  btnSelfServiceEnroll.addEventListener("click", errorWrap(enrollSelfService));

  initCommon();

  window.addEventListener('storage', reconcileSelfService);
  reconcileSelfService();
}

window.onload = init;


function hardwareKeyDialog() {
  return new Promise((resolve, _reject) => {
    let dialog = document.getElementById("dialog");
    dialog.showModal();
    ["abort", "continue", "enroll-other"].forEach((action) => {
      document.getElementById(`dialog--${action}`).onclick = () => {
        resolve(action);
        dialog.close();
      };
    });

    dialog.oncancel = () => {
      resolve("abort");
    };
  });
}

function removeKey(publicKey, enrollResult) {
  if (!PublicKeyCredential.signalUnknownCredential) {
    console.warn("could not remove key: browser does not implement signalUnknownCredential");
    return;
  }
  PublicKeyCredential.signalUnknownCredential({
    rpId: publicKey.rp.id,
    credentialId: enrollResult.id,
  }).then(res => {
    console.debug("key removal requested:", enrollResult.id, res);
  }).catch(e => {
    console.warn("could not remove key:", e);
  });
}
