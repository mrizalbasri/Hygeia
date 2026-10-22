// The Vite dev server does not always print its Local URL when stdout is not a
// TTY, so E2E tests must poll the HTTP endpoint instead of parsing log output.
export async function waitForServer(
	url,
	{ output = () => '', isAlive = () => true, timeout = 30000 } = {}
) {
	const deadline = Date.now() + timeout;
	while (Date.now() < deadline) {
		if (!isAlive()) break;
		try {
			const response = await fetch(url, { redirect: 'manual' });
			if (response.status < 500) return;
		} catch {
			// Connection refused while Vite is still booting.
		}
		await new Promise((resolve) => setTimeout(resolve, 200));
	}
	throw new Error(`Server did not become ready at ${url}. Output:\n${output()}`);
}
