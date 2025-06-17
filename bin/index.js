#!/usr/bin/env node

// axype cli
// @Darkceius
const yargs = require("yargs");
const { prompt } = require("enquirer");
const { hideBin } = require("yargs/helpers");
const colors = require("ansi-colors");
const symbols = colors.symbols;
const process = require("node:process");
const childProcess = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const os = require("node:os");

const homeDirectory = os.homedir();

const hasCLI = function (name) {
	const response = childProcess.spawnSync(name, ["-v"]);
	return response.error == undefined;
};

const getFolder = function () {
	const folder = path.join(homeDirectory, ".axype");
	if (!fs.existsSync(folder)) {
		fs.mkdirSync(folder);

		const files = ["secret"];
		files.forEach((v) => {
			const pth = path.join(folder, v);
			if (!fs.existsSync()) {
				fs.writeFileSync(pth, "");
			}
		});
	}

	return folder;
};

const getToken = function () {
	const folder = getFolder();
	const secret = path.join(folder, "secret");
	return fs.readFileSync(secret, "utf-8");
};

const setToken = function (data) {
	const folder = getFolder();
	const secret = path.join(folder, "secret");
	fs.writeFileSync(secret, data || "");
};

const init = async function () {
	if (!hasCLI("git")) {
		console.log(
			`${colors.magenta(symbols.cross)} Git is required to use this command!`
		);
		process.exit(1);
	}

	if (!hasCLI("rokit")) {
		console.log(
			`${colors.magenta(
				symbols.cross
			)} Rokit is not installed on this system!\n${colors.cyan(
				symbols.pointerSmall
			)} Download it here: https://github.com/rojo-rbx/rokit/releases/latest`
		);
		process.exit(1);
	}

	const data = await prompt([
		{
			type: "input",
			name: "name",
			message: "What will the project be named?",
		},
		{
			type: "confirm",
			name: "useGit",
			message: "Initialize git repository?",
			initial: true,
		},
		{
			type: "confirm",
			name: "useClient",
			message: "Add client-side support? (src/client)",
			initial: true,
		},
	]);

	const projectName = data.name.replace(/[^a-zA-Z0-9-]/g, "").trim();
	const useClient = data.useClient;

	let data2;
	if (useClient) {
		data2 = await prompt([
			{
				type: "confirm",
				name: "useClientLuau",
				message: "Use Luau syntax for client?",
				initial: false,
			},
		]);
	}

	const cwd = process.cwd();
	const targetPath = path.join(cwd, projectName);

	// confirmation prompt
	const confirmation = await prompt([
		{
			type: "confirm",
			name: "proceed",
			message: `Are you sure you want to initialize repository at ${colors.cyan(
				targetPath
			)}`,
			initial: true,
		},
	]);

	if (!confirmation.proceed) {
		console.log(`${colors.magenta(symbols.cross)} Cancelled!`);
		return;
	}

	// preventing overwriting
	if (fs.existsSync(targetPath)) {
		console.log(
			`${colors.magenta(
				symbols.cross
			)} Couldn't init because a file with the same target name already exists in this directory!`
		);
		process.exit(1);
	}

	console.log(
		`${colors.cyan(symbols.info)} Creating files! Might take some seconds...`
	);

	// cloning repo template
	{
		console.log(`${colors.cyan(symbols.info)} Cloning template repository...`);

		const gitURL = useClient
			? "https://github.com/axype/paste-template.git"
			: "https://github.com/axype/paste-template-noclient.git";

		try {
			const spawned = childProcess.spawnSync(
				"git",
				["clone", gitURL, projectName],
				{
					cwd,
				}
			);
			if (spawned.error) console.error(spawned.error);
			if (spawned.status !== 0) console.error(spawned.stderr.toString());
		} catch (err) {
			console.log(
				`${colors.magenta(
					symbols.cross
				)} Something went wrong while cloning the template repository!`
			);
			console.error(err);
			process.exit(1);
		}

		console.log(`${colors.green(symbols.check)} Successfully cloned!`);
	}

	console.log(`${colors.cyan(symbols.info)} Finishing up...`);

	// changing json files
	{
		const defaultProject = path.join(targetPath, "default.project.json");
		const contents = JSON.parse(fs.readFileSync(defaultProject, "utf-8"));
		contents["name"] = projectName;
		fs.writeFileSync(defaultProject, JSON.stringify(contents));
	}

	// use client luau syntax
	if (data2 && data2.useClientLuau) {
		fs.renameSync(
			path.join(targetPath, "src", "client", "init.lua"),
			path.join(targetPath, "src", "client", "init.luau")
		);
	}

	// initializing rokit
	childProcess.spawnSync("rokit", ["install"], { cwd: targetPath });
	console.log(`${colors.green(symbols.check)} Initialized Rokit!`);

	// .git related stuff
	{
		if (fs.existsSync(path.join(targetPath, ".git"))) {
			fs.rmSync(path.join(targetPath, ".git"), {
				force: true,
				recursive: true,
			});
		}

		const useGit = data.useGit;
		if (useGit) {
			childProcess.spawnSync("git", ["init"], { cwd: targetPath });
			childProcess.spawnSync("git", ["add", "."], {
				cwd: targetPath,
			});
			childProcess.spawnSync("git", ["commit", "-m", "Initial commit"], {
				cwd: targetPath,
			});
		}
	}

	console.log(`${colors.green(symbols.check)} Successfully created project!`);

	// open in vscode prompt
	{
		const response = await prompt([
			{
				type: "confirm",
				name: "openCode",
				message: "Open project in VS Code?",
				initial: false,
			},
		]);

		if (response.openCode) {
			childProcess.spawnSync("code", [".", "./src/server/init.luau"], {
				cwd: targetPath,
				shell: true,
			});
		}
	}
};

const publish = async function (name, sourcePath) {
	const file = path.join(process.cwd(), sourcePath);
	if (!fs.existsSync(file)) {
		console.log(
			`${colors.magenta(symbols.cross)} File at target path does not exist!`
		);
		process.exit(1);
	}

	let source;
	try {
		source = fs.readFileSync(file, "utf-8");
	} catch (err) {
		console.log(`${colors.magenta(symbols.cross)} Failed to read file!`);
		process.exit(1);
	}

	console.log(`${colors.cyan(symbols.info)} Publishing source...`);

	const data = await fetch(`https://axype.darkceius.dev/api/setSource`, {
		method: "POST",
		headers: {
			script: name,
			["Content-Type"]: "application/json",
			authentication: getToken(),
		},
		body: JSON.stringify({
			source: source,
		}),
	});

	const json = data.ok && (await data.json());

	if (data.ok && json.success) {
		console.log(
			`${colors.green(symbols.check)} Successfully updated source of ${name}!`
		);
	} else {
		console.log(`${colors.magenta(symbols.cross)} Failed to update source!`);
		console.log(colors.red(symbols.warning), await data.text());
		process.exit(1);
	}
};

yargs(hideBin(process.argv))
	.scriptName("axype")
	.usage("$0 <cmd> [args]")

	.command("init", "Initializes a template Axype project.", () => {}, init)

	.command(
		"set-token",
		"Updates your local Axype API token. It is used by the `publish` command.",
		() => {},
		async () => {
			const data = await prompt([
				{
					type: "password",
					name: "token",
					message: "API Token",
				},
			]);

			if (!data.token) return;

			setToken(data.token);
			console.log(`${colors.green(symbols.check)} Successfully updated token!`);
		}
	)

	.command(
		"remove-token",
		"Removes your local Axype API token.",
		() => {},
		() => {
			setToken(undefined);
			console.log(`${colors.green(symbols.check)} Removed token!`);
		}
	)

	.command(
		"publish <name> [path]",
		"Publishes the current paste source to the Axype API.",
		(yargs) => {
			return yargs
				.positional("name", {
					describe: "The paste name",
					type: "string",
				})
				.positional("path", {
					describe: "Path of paste's source",
					default: "output/server.luau",
					type: "string",
				});
		},
		(args) => publish(args.name, args.path)
	)

	.alias("st", "set-token")
	.alias("h", "help")
	.alias("i", "init")

	.demandCommand(1, "Command is required")

	.parse();
