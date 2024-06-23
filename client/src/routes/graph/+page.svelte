<script>
	const imgUrl = new URL('/dag.png', import.meta.url).href;
</script>

<svelte:head>
	<title>Images dependency graph</title>
</svelte:head>

<h1>Images dependency graph (Generated with graphviz)</h1>

<div>
	<img src={imgUrl} alt="Images dependency graph" />
</div>

<style>
	img {
		height: auto;
		max-width: 100%;
	}
</style>
