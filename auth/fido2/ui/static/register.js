async function enroll() {
  const token = document.getElementById("token").value;
  const resp = await fetch(
    `${baseURL}/internal/enroll/challenge?token=${token}`,
  );
  const respBody = await resp.json();
  console.log(respBody);

  const publicKey = {
    challenge: Uint8Array.from(respBody.data.challenge, (c) => c.charCodeAt(0)),
    rp: respBody.data.rp,
    user: {
      id: new Uint8Array([79, 252, 83, 72, 214, 7, 89, 26]),
      name: "jamiedoe",
      displayName: "Jamie Doe",
    },
    //attestation: "direct",
    pubKeyCredParams: [{ type: "public-key", alg: -7 }],
  };

  const enrollResult = await navigator.credentials.create({ publicKey });
  console.log(enrollResult);
  console.log(JSON.stringify(enrollResult));
  console.log(JSON.stringify(enrollResult.toJSON()));

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
}

function init() {
  console.log("init");
  const btnEnroll = document.getElementById("btn-enroll");
  btnEnroll.addEventListener("click", enroll);
}

console.log("register init callback");
window.onload = init;
