async function listKeys() {
  const selfServiceInfo = getSelfServiceInfo();

  const resp = await checkedFetch(`${baseURL}/self-service/keys`, {
    method: "LIST",
    headers: {
      "X-Vault-Token": selfServiceInfo.token,
    },
  });
  const respBody = await resp.json();
  const details = respBody.data.details;
  const divKeysList = document.getElementById("keys-list");
  divKeysList.innerHTML = "";

  if (!details) {
    divKeysList.innerHTML =
      "no keys found for current user <a href='./register.html'><button class='button-primary'>Enroll now</button></a>";
    return;
  }

  Object.entries(details).forEach(([key, data]) => {
    const element = document.createElement("div");
    element.className = "key";
    divKeysList.appendChild(element);

    const nameDiv = document.createElement("div");
    nameDiv.innerText = data.name;
    element.appendChild(nameDiv);

    const btnRename = document.createElement("button");
    btnRename.innerText = "rename";
    btnRename.addEventListener(
      "click",
      renameKey.bind(undefined, key, data.name),
    );
    element.appendChild(btnRename);

    const btnDelete = document.createElement("button");
    btnDelete.innerText = "delete";
    btnDelete.addEventListener("click", deleteKey.bind(undefined, key));
    element.appendChild(btnDelete);
  });
}

async function deleteKey(id) {
  const selfServiceInfo = getSelfServiceInfo();

  await checkedFetch(`${baseURL}/self-service/keys/` + id, {
    method: "DELETE",
    headers: {
      "X-Vault-Token": selfServiceInfo.token,
    },
  });

  await listKeys();
}

async function renameKey(id, old) {
  let newName = prompt(`new name for key with id ${id}`, old);
  if (!newName || newName === old) {
    return;
  }

  const selfServiceInfo = getSelfServiceInfo();

  await checkedFetch(`${baseURL}/self-service/keys/${id}`, {
    method: "PATCH",
    body: JSON.stringify({
      name: newName,
    }),
    headers: {
      "X-Vault-Token": selfServiceInfo.token,
      "Content-Type": "application/merge-patch+json",
    },
  });

  await listKeys();
}

function init() {
  listKeys();
}

window.onload = init;
