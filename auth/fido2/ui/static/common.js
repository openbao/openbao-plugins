function findAPI() {
  let parts = window.location.href.split("/");
  
  while (parts[parts.length-1] != "internal" && parts.length > 3) {
    parts.pop();
  }

  parts.pop(); // pop the "internal"

  return parts.join("/");
}

const baseURL = findAPI();
