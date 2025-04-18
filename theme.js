fetch ("http://localhost:8080/api/data",{
    method:"POST",
    headers:{
        "Content-Type":"application/json"
    },
    body:JSON.stringify({})
})

.then(response => {
    if (!response.ok) {
        return response.text().then(text => {
            throw new Error("Réponse invalide du serveur : " + text);
        });
    }
    return response.text(); 
})

.then (data => {
    const theme = data.theme;
    let themeswitch = 1;

    if (!theme || !themeswitch){
        document.body.classList.add("dark-mode");
    }else if (theme||themeswitch) {
        document.body.classList.remove("dark-mode");
    }
})

.catch(error =>{
    console.error("Error resevoire des data depuis le back:\n",error);
})