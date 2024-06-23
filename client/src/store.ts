import { writable } from 'svelte/store';

// Report global data
export const reportsStore = writable([]);

// Images data (build logs, tests & scan results)
export const imagesStore = writable([]);
