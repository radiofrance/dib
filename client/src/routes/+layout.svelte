<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { imagesStore, reportDataStore } from '$stores/report.ts';

	onMount(() => {
		// "dib_images" is set from included "map.js" generated file
		reportDataStore.set(window.dib_images);
		$reportDataStore.forEach(async (imageName) => {
			const buildLogs = await fetchImageData(imageName, 'docker.txt');
			const testsLogs = await fetchImageData(imageName, 'goss.json');
			const scanLogs = await fetchImageData(imageName, 'trivy.json');

			imagesStore.set([
				...$imagesStore,
				{
					name: imageName,
					docker: buildLogs,
					goss: testsLogs,
					trivy: scanLogs
				}
			]);
		});
	});

	async function fetchImageData(name: string, file: string) {
		try {
			const response = await fetch(`/data/${name}/${file}`);
			if (!response.ok) {
				throw new Error(`${response.status} ${response.statusText}`);
			}
			if (file === 'docker.txt') {
				return await response.text();
			}
			return await response.json();
		} catch (error: unknown) {
			return `There has been a problem fetching ${file}: ${error.message}`;
		}
	}
</script>

<div class="layout">
	<div class="navbar">
		<img src="./logo.png" alt="dib logo" />
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
