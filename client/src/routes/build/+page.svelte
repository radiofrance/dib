<script lang="ts">
	import { imagesStore } from '$stores/report.ts';
	import Accordion from '$lib/components/ui/accordion/Accordion.svelte';
</script>

<svelte:head>
	<title>Images build logs</title>
</svelte:head>

<h1>Images build logs</h1>

<div>
	{#each $imagesStore as image}
		<Accordion headerText={image.name}>
			<div class="code">
				<pre><code>
{image.docker}
				</code></pre>
			</div>
		</Accordion>
	{/each}
</div>

<style>
	.code {
		margin: 0 0 1rem 0;
		padding: 0 1rem;
		color: #d8d8d8;
		background: #181818;
	}

	.code pre {
		margin: 0;
	}
</style>
