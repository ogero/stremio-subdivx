import {Heart} from "lucide-react";
import LanguagePicker from "@/components/LanguagePicker.tsx";
import {withTranslation} from "react-i18next";
import Logo from "@/components/Logo.tsx";
import {TFunction} from 'i18next';

type FooterProps = {
  t: TFunction
}

const Footer = ({t}: FooterProps) => {
  return (
    <footer className="bg-black border-t border-gray-800">
      <div className="container mx-auto px-4 py-12">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
          <div className="col-span-1 md:col-span-2">
            <Logo className="mb-4"/>
            <p className="text-gray-400 leading-relaxed max-w-md">
              {t('Disclaimer')}
            </p>
            <p className="text-gray-400 leading-relaxed max-w-md">
              {t('Use at your own risk.')}
            </p>
          </div>

          <div>
            <h3 className="text-white font-semibold mb-4">Etc.</h3>
            <ul className="space-y-2">
              <li><LanguagePicker/></li>
              <li><a href={'https://www.stremio.com/'} rel={'noopener'} target={'_blank'} className="text-gray-400 hover:text-white transition-colors">Stremio.com</a></li>
              <li><a href={'https://github.com/ogero/stremio-subdivx'} target={'_blank'} className="text-gray-400 hover:text-white transition-colors">Github</a></li>
              <li><a href={'https://cafecito.app/ogero'} rel={'noopener'} target={'_blank'} className="text-gray-400 hover:text-white transition-colors">{t('Donate')}</a></li>
            </ul>
          </div>

        </div>

        <div className="border-t border-gray-800 mt-12 pt-8 flex flex-col md:flex-row items-center justify-between">
          <p className="text-gray-400 text-sm">
            &nbsp;
          </p>
          <div className="flex items-center space-x-1 text-gray-400 text-sm mt-4 md:mt-0">
            <span>{t('Made with')}</span>
            <Heart size={16} className="text-red-500 fill-current"/>
            <span>{t('for the Stremio community')}</span>
          </div>
        </div>
      </div>
    </footer>
  );
};


export default withTranslation()(Footer);
