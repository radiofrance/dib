<script lang="ts">
	import { reportDataStore } from '$stores/report.ts';
	import Accordion from '$lib/components/ui/accordion/Accordion.svelte';
</script>

<svelte:head>
	<title>Report Overview | DIB</title>
</svelte:head>

<h1>Report Overview</h1>

{#each $reportDataStore as image}
	<Accordion headerText={image}>lorem</Accordion>
{/each}
