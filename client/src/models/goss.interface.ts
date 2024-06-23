export interface GossResults {
	name: string;
	errors: string;
	tests: string;
	failures: string;
	skipped: string;
	time: string;
	timestamp: string;
	testcases: Testcase[];
}

export interface Testcase {
	class_name: string;
	file: string;
	name: string;
	time: string;
	failure?: string;
	system_out?: string;
}
