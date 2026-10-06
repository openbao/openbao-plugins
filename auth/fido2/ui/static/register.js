async function enrollToken() {
  return await enrollCommon(document.getElementById("token").value);
}
async function enrollCommon(token) {
  const resp = await checkedFetch(
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
  if (publicKey.excludeCredentials) {
    publicKey.excludeCredentials.forEach((credential) => {
      credential.id = Uint8Array.fromBase64(credential.id, {
        alphabet: "base64url",
      });
    });
  }

  const enrollResult = await navigator.credentials.create({ publicKey });

  const assertionResp = await checkedFetch(`${baseURL}/internal/enroll/assertion`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      // fidoToJSON is needed, because of this bug: https://github.com/microsoft/playwright/issues/43093
      // should be fixed in the next release of playwright
      assertion: JSON.stringify(fidoToJSON(enrollResult)),
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

  localStorage.setItem(localStorageKey("alias"), auth.data.alias);

  lastAuth = auth;
}

async function enrollSelfService() {
  const selfServiceInfo = getSelfServiceInfo();
  if (!selfServiceInfo) {
    throw new Error("not logged in");
  }

  const resp = await checkedFetch(
    `${baseURL}/self-service-enroll`, {
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

/** @typedef {object} tokenInfo
 * @property {string} userRootNamespace
 * @property {string} displayName
 * @property {object} backend
 * @property {string} backend.type
 * @property {string} backend.typeDisplay
 * @property {string} backend.description
 * @property {string} backend.tokenPath
 * @property {string} backend.displayNamePath
 * @property {string[]} backend.formAttributes
 * @property {string} token
 * @property {string[]} policies
 * @property {boolean} renewable
 * @property {string} entity_id
 * @property {number} ttl
 * @property {number} tokenExpirationEpoch
 *
 * Example:
    {
      "userRootNamespace": "",
      "displayName": "alice",
      "backend": {
        "type": "userpass",
        "typeDisplay": "Username",
        "description": "A simple username and password backend.",
        "tokenPath": "client_token",
        "displayNamePath": "metadata.username",
        "formAttributes": [
          "username",
          "password"
        ]
      },
      "token": "s.token",
      "policies": [
        "default",
        "read-kv"
      ],
      "renewable": true,
      "entity_id": "31ae1911-d3cc-f5f1-d7a3-8e60edb55773",
      "ttl": 2764800,
      "tokenExpirationEpoch": 1793347030958
    }
 *
 * TODO: would be nice to use TypeScript here instead
 */

/**
 * @return {tokenInfo}
 */
function getSelfServiceInfo() {
  const selectedAuth = localStorage.getItem('selectedAuth');
  if (!selectedAuth) {
    console.debug('self service not available: "selectedAuth" not set');
    return null;
  }

  const key = `${TOKEN_PREFIX}${selectedAuth}${TOKEN_SEPARATOR}1`;
  const dataJSON = localStorage.getItem(key);
  if (!dataJSON) {
    console.debug(`self service not available: "${key}" not set`);
    return null;
  }

  return JSON.parse(dataJSON);
}
