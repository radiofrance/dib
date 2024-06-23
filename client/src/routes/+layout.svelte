<script>
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { reportsStore, imagesStore } from '../store.ts';

	onMount(() => {
		// "dib_images" is set from included "map.js" generated file
		// no-undef
		reportsStore.set(window.dib_images);
		$reportsStore.forEach(async (imageName) => {
			const buildLogs = await fetchImageData(imageName, 'docker.txt');
			const scanLogs = await fetchImageData(imageName, 'trivy.json');
			const testsLogs = await fetchImageData(imageName, 'goss.json');

			imagesStore.set([
				...$imagesStore,
				{
					name: imageName,
					docker: buildLogs,
					tests: testsLogs,
					scan: scanLogs
				}
			]);
		});
	});

	async function fetchImageData(name, file) {
		try {
			const response = await fetch(`/data/${name}/${file}`);
			if (!response.ok) {
				throw new Error(`${response.status} ${response.statusText}`);
			}
			if (file === 'docker.txt') {
				return await response.text();
			}
			return await response.json();
		} catch (error) {
			return `There has been a problem fetching ${file}: ${error.message}`;
		}
	}
</script>

<div class="layout">
	<div class="navbar">
		<img src="./dib.png" alt="dig_logo" />
		<ul>
			<li>
				<a href="/" class:current={$page.url.pathname === '/'}>Overview</a>
			</li>
			<li>
				<a href="/graph" class:current={$page.url.pathname.startsWith('/graph')}>Graph</a>
			</li>
			<li>
				<a href="/build" class:current={$page.url.pathname.startsWith('/build')}>Build Logs</a>
			</li>
			<li>
				<a href="/test" class:current={$page.url.pathname.startsWith('/test')}>Tests logs</a>
			</li>
			<li>
				<a href="/scan" class:current={$page.url.pathname.startsWith('/scan')}>Scan logs</a>
			</li>
		</ul>
	</div>
	<div class="content">
		<slot />
	</div>
</div>

<style>
	:global(body) {
		margin: 0;
		padding: 0;
	}

	.layout {
		display: flex;
		flex-direction: row;
		height: 100%;
		min-height: 100vh;
		width: auto;
		align-items: stretch;
	}

	.navbar {
		flex: 0 0 auto;
		background-color: #151f29;
		padding: 0.5rem;
	}

	.navbar ul {
		list-style-type: none;
		text-align: left;
	}

	.navbar ul li a {
		text-decoration: none;
		color: white;
	}

	.navbar ul li a:hover {
		color: grey;
	}

	.navbar ul li .current {
		color: grey;
	}

	.content {
		width: 100%;
		padding: 1rem 2rem;
		background-color: #e3e9f7;
	}
</style>
