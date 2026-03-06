import image1 from "../assets/zadig/zadig1.jpg"
import image2 from "../assets/zadig/zadig2.jpg"
import image3 from "../assets/zadig/zadig3.jpg"

export function zadigSteps(printerName) {
    return [
        {
            title: 'Download Zadig',
            desc: 'Download the latest version of Zadig from the official website.',
            link: 'https://zadig.akeo.ie',
            linkLabel: 'Open Zadig website',
        },
        {
            title: 'Open Zadig',
            desc: 'Run Zadig.exe as Administrator.\nGo to Options → List All Devices.',
            image: image1,
        },
        {
            title: 'Select your printer',
            desc: `Find the printer in the dropdown list.\n The name may vary, but it should look similar to "${printerName}".`,
            image: image2,
        },
        {
            title: 'Install WinUSB driver',
            desc: 'Make sure WinUSB is selected, then click "Replace Driver". Wait for completion.',
            image: image3,
        },
    ]
}

export function brewSteps(printerName) {
    return [
        {
            title: 'Install Homebrew',
            desc: 'Homebrew is a package manager for macOS. If you already have it, skip to the next step.',
            link: 'https://brew.sh',
            linkLabel: 'Open Homebrew website',
        },
        {
            title: 'Install libusb',
            desc: 'Once Homebrew is installed, run this command in Terminal:',
            codes: ['brew install libusb'],

        },
        {
            title: 'Reconnect your printer',
            desc: `Unplug and replug your printer "${printerName}".\nThe driver should now be detected automatically.`,
        },
    ]
}

export function linuxSteps(printerName) {
    return [
        {
            title: 'Install libusb',
            desc: 'Open a terminal and run the command for your distribution:',
            codes: ['# Debian / Ubuntu\nsudo apt-get install libusb-1.0-0',
                '# Fedora / RHEL\nsudo dnf install libusb1',
                '# Arch\nsudo pacman -S libusb'],
        },
        {
            title: 'Reconnect your printer',
            desc: `Unplug and replug your printer "${printerName}".\nThe driver should now be detected automatically.`,
        },
    ]
}

