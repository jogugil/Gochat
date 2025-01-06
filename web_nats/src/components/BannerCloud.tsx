import React, { useState } from 'react';
import { handleBannerClick, handleMouseEnter } from './banners';

interface BannerCloudProps {
  imageUrl: string;
  titleSlogan: string;    // Prop para el título del eslogan
  subtitleSlogan: string; // Prop para el subtítulo del eslogan
}

const BannerCloud: React.FC<BannerCloudProps> = ({ imageUrl, titleSlogan, subtitleSlogan }) => {
  const [isHovered, setIsHovered] = useState(false);

  return (
    <div
      className="banner-container"
      onClick={() => handleBannerClick('Cloud Computing')}
      onMouseEnter={() => {
        handleMouseEnter('Cloud Computing');
        setIsHovered(true); // Mostrar los iconos al pasar el mouse
      }}
      onMouseLeave={() => setIsHovered(false)} // Ocultar los iconos cuando el mouse sale
    >
      <div className="banner-content">
        {/* Imagen del banner */}
        <div className="banner-image-container">
          <img src={imageUrl} alt="Banner de Cloud Computing" className="banner-image" />
        </div>

        {/* Slogan debajo de la imagen, usando props */}
        <div className="banner-slogan">
          <h2>{titleSlogan}</h2>
          <p>{subtitleSlogan}</p>
        </div>

        {/* Si el mouse está encima, mostrar los iconos centrados */}
        {isHovered && (
          <div className="banner-summary">
            <div className="tech-icons">
              <img src="https://es.m.wikipedia.org/wiki/Archivo:Amazon_Web_Services_Logo.svg" alt="AWS" className="icon" />
              <img src="https://logodownload.org/wp-content/uploads/2021/06/google-cloud-logo-0.png" alt="Google Cloud" className="icon" />
              <img src="https://upload.wikimedia.org/wikipedia/commons/f/fa/Microsoft_Azure.svg" alt="Azure" className="icon" />
            </div>
          </div>
        )}

        {/* Pie de imagen con copyright */}
        <div className="banner-footer">
          <small>© SmartIAService's - José Javier Gutiérrez Gil</small>
        </div>
      </div>
    </div>
  );
};

export default BannerCloud;
