const canvas = document.getElementById("renderCanvas");
const engine = new BABYLON.Engine(canvas, true);
const scene = new BABYLON.Scene(engine);
scene.clearColor = new BABYLON.Color3(0.1, 0.1, 0.1);

const camera = new BABYLON.ArcRotateCamera("camera", -Math.PI/2, Math.PI/2.5, 3, BABYLON.Vector3.Zero(), scene);
camera.attachControl(canvas, true);

const light = new BABYLON.HemisphericLight("light", new BABYLON.Vector3(1, 1, 0), scene);

fetch("http://localhost:8080/api/data", {
  method: "POST",
  headers: {
    "Content-Type": "application/json"
  },
  body: JSON.stringify({})
})
.then(async res => {
    const text = await res.text();
    if (!text) throw new Error("Réponse vide du serveur");
    return JSON.parse(text);
  })
  
.then(data => {
  document.getElementById("product-name").innerText = data.name;
  document.getElementById("product-description").innerText = data.description;
  document.getElementById("product-image").src = data.image;
  document.getElementById("product-price").innerText = `${data.price} €`;
  document.getElementById("product-rating").innerText = `⭐ ${data.rating} (${data.reviews} avis)`;

  if (data.theme === "dark") {
    document.body.classList.add("dark");
  }

  // Load 3D model
  BABYLON.SceneLoader.Append("http://localhost:8080/assets/", data.mesh, scene, function () {
    scene.createDefaultCameraOrLight(true, true, true);
  });
})
.catch(err => {
  console.error("Erreur fetch backend:", err);
});

engine.runRenderLoop(() => {
  scene.render();
});

window.addEventListener("resize", () => {
  engine.resize();
});
