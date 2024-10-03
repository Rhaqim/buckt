function formatFileName(category) {
	let fileName = category.toLowerCase();
	// let formattedFileName = fileName.replace(/[\s.\-\/]+/g, '');
	let formattedFileName = fileName.replace(/[\s.\-\/']+/g, "");

	console.log(formattedFileName);

	storyTitle = category;

	return formattedFileName;
}

formatFileName("The Necromancer's Lair");