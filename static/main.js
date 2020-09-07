var deleteElms = document.querySelectorAll("span.delete-list")
for (let i = 0; i < deleteElms.length; i++) {
    deleteElms.item(i).onclick = e => {
        let xhr = new XMLHttpRequest()
        xhr.onreadystatechange = () => {
            if (xhr.readyState == XMLHttpRequest.DONE && xhr.status >= 200 && xhr.status < 300) {
                window.location.reload()
            }
        }
        xhr.open("DELETE", `/lists?name=${e.target.dataset.name}`)
        xhr.send()
    }
}

var deleteSubEls = document.querySelectorAll("span.delete-subscriber")
for (let i = 0; i < deleteSubEls.length; i++) {
    deleteSubEls[i].onclick = e => {
        let xhr = new XMLHttpRequest()
        xhr.onreadystatechange = () => {
            if (xhr.readyState == XMLHttpRequest.DONE && xhr.status >= 200 && xhr.status < 300) {
                window.location.reload()
            }
        }
        xhr.open("DELETE", `${window.location.pathname}?email=${e.target.dataset.email}`)
        xhr.send()
    }
}
