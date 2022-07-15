function toggleMenu() {
	let menu = document.getElementById("dropdown-menu");
	if (menu.style.display === "none" || menu.style.display === "") {
		menu.style.display = "block";
	} else {
		menu.style.display = "none";
	}
}
