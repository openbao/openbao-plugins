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

function persistToken() {
  if (lastAuth === null) {
    throw new Error("not logged in");
  }
  localStorage.setItem(`${TOKEN_PREFIX}passkey${TOKEN_SEPARATOR}1`, JSON.stringify({
    token:     lastAuth.auth.client_token,
    entity_id: lastAuth.auth.entity_id,
    policies:  lastAuth.auth.policies,
    renewable: lastAuth.auth.renewable,
  }))
}

function openUI() {
  persistToken();
  window.location.pathname = "/ui";
}
